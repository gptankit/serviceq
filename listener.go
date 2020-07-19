package main

import (
	_ "context"
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"github.com/gptankit/serviceq/model"
	_ "golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
	"net"
	"time"
)

// getListener returns a new http(s) listener.
func getListener(sqp model.ServiceQProperties) (net.Listener, error) {

	fmt.Println(sqp.SSLEnabled)
	fmt.Println(sqp.SSLAutoEnabled)
	fmt.Println(sqp.SSLAutoCertificateDir)
	fmt.Println(sqp.SSLAutoEmail)
	fmt.Println(sqp.SSLAutoDomains)
	fmt.Println(sqp.SSLAutoRenewBefore)

	transport := "tcp"
	addr := ":" + sqp.ListenerPort
	certificate := sqp.SSLCertificateFile
	key := sqp.SSLPrivateKeyFile

	if !sqp.SSLEnabled {
		return newListener(transport, addr)
	}

	if sqp.SSLAutoEnabled {
		fmt.Println("creating tls auto listener")
		return newListener(transport, addr, applyTLSAuto(sqp.SSLAutoCertificateDir, sqp.SSLAutoEmail, sqp.SSLAutoDomains, sqp.SSLAutoRenewBefore))
	} else {
		fmt.Println("creating tls standard listener")
		return newListener(transport, addr, applyTLS(certificate, key))
	}
}

// newListener creates a new net.Listener object.
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

// applyTLS upgrades non-TLS listener to TLS listener.
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

func applyTLSAuto(certDir string, email string, domains string, renewBefore int32) func(*net.Listener) error {

	return func(l *net.Listener) error {

		certManager := autocert.Manager{
			Prompt:      autocert.AcceptTOS,
			Cache:       autocert.DirCache(certDir),
			HostPolicy:  autocert.HostWhitelist(domains),
			Email:       email,
			RenewBefore: time.Duration(renewBefore) * time.Hour * 24,
		}

		tlsConfig := &tls.Config{
			GetCertificate: certManager.GetCertificate,
			ServerName:     "serviceq",
			NextProtos:     []string{"http-01", "http/1.1", "http/1.0"},
			Time:           time.Now,
			Rand:           rand.Reader,
		}
		tlsConfig.PreferServerCipherSuites = true
		*l = tls.NewListener(*l, tlsConfig)
		return nil
	}
}
