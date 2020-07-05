package main

import (
	"crypto/rand"
	"crypto/tls"
	"github.com/gptankit/serviceq/model"
	"net"
	"time"
)

func getListener(sqp model.ServiceQProperties) (net.Listener, error) {

	transport := "tcp"
	addr := ":" + sqp.ListenerPort
	certificate := sqp.SSLCertificateFile
	key := sqp.SSLPrivateKeyFile

	if !sqp.SSLEnabled {
		return newListener(transport, addr)
	} else {
		return newListener(transport, addr, applyTLS(certificate, key))
	}
}

func newListener(transport string, addr string, options ...func(*net.Listener) error) (net.Listener, error) {

	listener, err := net.Listen(transport, addr)
	if err != nil {
		return listener, err
	}

	for _, option := range options {
		err = option(&listener)
		if err != nil {
			return listener, err // further options won't be executed
		}
	}

	return listener, nil
}

func applyTLS(certificate string, key string) func(*net.Listener) error {

	return func(l *net.Listener) error {

		cert, err := tls.LoadX509KeyPair(certificate, key)
		if err != nil {
			return err
		}

		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{cert},
			ServerName:   "serviceq",
			NextProtos:   []string{"http/1.1", "http/1.0"},
			Time:         time.Now,
			Rand:         rand.Reader,
		}
		tlsConfig.BuildNameToCertificate()
		tlsConfig.PreferServerCipherSuites = true

		*l = tls.NewListener(*l, tlsConfig)
		return nil
	}
}
