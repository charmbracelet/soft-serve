package web

import (
	"crypto/tls"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/charmbracelet/log"
)

// CertReloader is responsible for reloading TLS certificates when a SIGHUP signal is received.
type CertReloader struct {
	certMu   sync.RWMutex
	cert     *tls.Certificate
	certPath string
	keyPath  string
}

// NewCertReloader creates a new CertReloader that watches for SIGHUP signals.
func NewCertReloader(certPath, keyPath string, logger *log.Logger) (*CertReloader, error) {
	reloader := &CertReloader{
		certPath: certPath,
		keyPath:  keyPath,
	}

	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, err
	}
	reloader.cert = &cert

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGHUP)
		for range sigChan {
			logger.Info("attempting to reload TLS certificate and key", "cert", certPath, "key", keyPath)
			if err := reloader.maybeReload(); err != nil {
				logger.Error("failed to reload TLS certificate, keeping old certificate", "err", err)
			}
		}
	}()

	return reloader, nil
}

// maybeReload attempts to reload the certificate and key.
func (cr *CertReloader) maybeReload() error {
	newCert, err := tls.LoadX509KeyPair(cr.certPath, cr.keyPath)
	if err != nil {
		return err
	}

	cr.certMu.Lock()
	defer cr.certMu.Unlock()
	cr.cert = &newCert
	return nil
}

// GetCertificateFunc returns a function that can be used with tls.Config.GetCertificate.
func (cr *CertReloader) GetCertificateFunc() func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	return func(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {
		cr.certMu.RLock()
		defer cr.certMu.RUnlock()
		return cr.cert, nil
	}
}
