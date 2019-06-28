package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"github.com/davecgh/go-spew/spew"
	//quic "github.com/lucas-clemente/quic-go"
	//"bufio"
	quicconn "github.com/marten-seemann/quic-conn"
	"go.uber.org/zap"
	"math/big"
	"net"
	"quictun/logger"
	"sync"
	"time"
)

var Logger *zap.SugaredLogger

func main() {
	Logger = logger.InitLog()

	quicAddr := flag.String("quicAddr", "localhost:4343", "quicHost")
	tcpAddr := flag.String("tcpAddr", "localhost:1935", "tapHost")
	flag.Parse()

	//listener, err := quic.ListenAddr(*quicAddr, generateTLS(), nil)
	listener, err := quicconn.Listen("udp", *quicAddr, generateTLS())
	if err != nil {
		Logger.Errorf("quic listen failed: %v", err)
		return
	}

	Logger.Infof("quic listen %s success.", *quicAddr)

	for {
		if qconn, err := listener.Accept(); err == nil {
			if err != nil {
				Logger.Errorf("quic accept error: ", err)
				return
			}

			Logger.Infof("quic accept success")
			go handleQuicConn(qconn, *tcpAddr)

		} else {
			Logger.Errorf("quic accept fail:", err)
		}
	}
}

func handleQuicConn(qconn net.Conn, tcpAddr string) {
	defer qconn.Close()

	tconn, err := net.DialTimeout("tcp", tcpAddr, time.Duration(10)*time.Second)
	if err != nil {
		Logger.Errorf("dial tcp error: ", err)
		return
	}

	defer tconn.Close()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, 81920)
		count := 0
		for {
			n, err := qconn.Read(buf)
			if err != nil {
				Logger.Errorf("quic conn read error: ", err)
				return
			}
			count++
			if count > 5 {
				return
			}
			
			spew.Dump(buf[:n])
			_, err = tconn.Write(buf[:n])
			if err != nil {
				Logger.Errorf("tcp conn write error: ", err)
				return
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, 81920)
		for {
			n, err := tconn.Read(buf)
			if err != nil {
				Logger.Errorf("tcp conn read error: ", err)
				return
			}

			_, err = qconn.Write(buf[:n])
			if err != nil {
				Logger.Errorf("quic conn write error: ", err)
				return
			}
		}
	}()

	wg.Wait()
}

func generateTLS() *tls.Config {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		panic(err)
	}

	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		panic(err)
	}

	return &tls.Config{Certificates: []tls.Certificate{tlsCert}}
}
