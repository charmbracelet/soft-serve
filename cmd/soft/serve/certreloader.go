package serve

import (
	"crypto/tls"
	"sync"

	"charm.land/log/v2"
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

	return reloader, nil
}

// Reload attempts to reload the certificate and key.
func (cr *CertReloader) Reload() error {
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
