package main

import (
	"crypto/rand"
	"crypto/tls"
	"net"
	"time"

	"github.com/gptankit/serviceq/model"
	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
)

// ListenerOption is used for extending default listener
type ListenerOption func(*net.Listener) error

// newListener returns a new http(s) listener
func newListener(sqp *model.ServiceQProperties) (*net.Listener, error) {

	transport := "tcp"
	addr := ":" + sqp.ListenerPort
	certificate := sqp.SSLCertificateFile
	key := sqp.SSLPrivateKeyFile

	if !sqp.SSLEnabled {
		return getListener(transport, addr)
	}

	if sqp.SSLAutoEnabled {
		return getListener(transport, addr, withTLSAuto(sqp.SSLAutoCertificateDir, sqp.SSLAutoEmail, sqp.SSLAutoDomains, sqp.SSLAutoRenewBefore))
	} else {
		return getListener(transport, addr, withTLS(certificate, key))
	}
}

// getListener creates a new net.Listener object
func getListener(transport string, addr string, listenerOptions ...ListenerOption) (*net.Listener, error) {

	listener, err := net.Listen(transport, addr)
	if err != nil {
		return &listener, err
	}

	for _, listenerOption := range listenerOptions {
		err = listenerOption(&listener)
		if err != nil {
			return &listener, err // further options won't be executed
		}
	}

	return &listener, nil
}

// withTLS upgrades non-TLS listener to TLS listener with user-provided ssl certificate and key
func withTLS(certificate string, key string) ListenerOption {

	return func(l *net.Listener) error {

		cert, err := tls.LoadX509KeyPair(certificate, key)
		if err != nil {
			return err
		}

		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{cert},
			ServerName:   "serviceq",
			NextProtos:   []string{"http/1.1", "http/1.0"},
			MinVersion:   tls.VersionTLS12,
			Time:         time.Now,
			Rand:         rand.Reader,
		}
		tlsConfig.BuildNameToCertificate()
		tlsConfig.PreferServerCipherSuites = true

		*l = tls.NewListener(*l, tlsConfig)
		return nil
	}
}

// withTLSAuto upgrades non-TLS listener to TLS listener with automatic ssl management
func withTLSAuto(certDir string, email string, domains string, renewBefore int32) ListenerOption {

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
			NextProtos:     []string{"http/1.1", "http/1.0", acme.ALPNProto},
			MinVersion:     tls.VersionTLS12,
			Time:           time.Now,
			Rand:           rand.Reader,
		}
		tlsConfig.PreferServerCipherSuites = true
		*l = tls.NewListener(*l, tlsConfig)
		return nil
	}
}
