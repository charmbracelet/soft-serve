//go:build unix

package web

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"os"
	"syscall"
	"testing"
	"time"

	"charm.land/log/v2"
)

func generateTestCert(t *testing.T, certPath, keyPath, cn string) {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	template := x509.Certificate{
		SerialNumber: nil,
		Subject: pkix.Name{
			CommonName: cn,
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(time.Hour),
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		t.Fatal(err)
	}

	certFile, err := os.Create(certPath)
	if err != nil {
		t.Fatal(err)
	}
	defer certFile.Close()

	pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: certBytes})

	keyFile, err := os.Create(keyPath)
	if err != nil {
		t.Fatal(err)
	}
	defer keyFile.Close()

	pem.Encode(keyFile, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})
}

func TestCertReloader(t *testing.T) {
	dir := t.TempDir()
	certPath := dir + "/cert.pem"
	keyPath := dir + "/key.pem"

	// Initial cert
	generateTestCert(t, certPath, keyPath, "cert-v1")

	logger := log.New(os.Stderr)

	certReloader, err := NewCertReloader(certPath, keyPath, logger)
	if err != nil {
		t.Fatalf("failed to create reloader: %v", err)
	}

	getCert := certReloader.GetCertificateFunc()

	cert1, err := getCert(nil)
	if err != nil {
		t.Fatal(err)
	}

	// Replace cert on disk
	generateTestCert(t, certPath, keyPath, "cert-v2")

	// Trigger reload
	if err := syscall.Kill(os.Getpid(), syscall.SIGHUP); err != nil {
		t.Fatalf("failed to send SIGHUP: %v", err)
	}

	// Allow async goroutine to reload
	time.Sleep(100 * time.Millisecond)

	cert2, err := getCert(nil)
	if err != nil {
		t.Fatal(err)
	}

	if cert1 == cert2 {
		t.Fatal("certificate was not reloaded after SIGHUP")
	}
}
