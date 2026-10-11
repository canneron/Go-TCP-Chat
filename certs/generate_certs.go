package certs

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func GenerateCSR(host, port string) ([]byte, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("generating key: %w", err)
	}

	address := fmt.Sprintf("%s:%s", host, port)
	ip := net.ParseIP(strings.Trim(host, "[]"))

	var dnsNames []string
	var ipAddresses []net.IP

	if ip != nil {
		ipAddresses = append(ipAddresses, ip)
	} else {
		dnsNames = append(dnsNames, host)
	}

	csrTemplate := x509.CertificateRequest{
		Subject: pkix.Name{
			Organization: []string{"CC"},
			CommonName:   address,
		},
		DNSNames:    dnsNames,
		IPAddresses: ipAddresses,
	}

	der, err := x509.CreateCertificateRequest(rand.Reader, &csrTemplate, key)
	if err != nil {
		return nil, fmt.Errorf("creating CSR: %w", err)
	}

	if err := writeKey(key, KeyPath(port)); err != nil {
		return nil, err
	}

	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: der}), nil
}

func SignCSR(csrPEM []byte, caCert *x509.Certificate, caKey *rsa.PrivateKey) ([]byte, error) {
	block, _ := pem.Decode(csrPEM)
	if block == nil || block.Type != "CERTIFICATE REQUEST" {
		return nil, errors.New("no CSR PEM data")
	}

	csr, err := x509.ParseCertificateRequest(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parsing CSR: %w", err)
	}

	if err := csr.CheckSignature(); err != nil {
		return nil, fmt.Errorf("invalid CSR signature: %w", err)
	}

	if len(csr.DNSNames)+len(csr.IPAddresses) == 0 {
		return nil, errors.New("CSR has no DNS or IP SANs")
	}

	serialNum, err := newSerialNumber()
	if err != nil {
		return nil, fmt.Errorf("generating serial num: %w", err)
	}

	certTemplate := x509.Certificate{
		SerialNumber: serialNum,
		Subject:      csr.Subject,
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),

		KeyUsage: x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
			x509.ExtKeyUsageClientAuth,
		},
		BasicConstraintsValid: true,
		DNSNames:              csr.DNSNames,
		IPAddresses:           csr.IPAddresses,
	}

	der, err := x509.CreateCertificate(rand.Reader, &certTemplate, caCert, csr.PublicKey, caKey)
	if err != nil {
		return nil, fmt.Errorf("signing cert: %w", err)
	}

	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), nil
}

func SaveCert(port string, certPEM []byte) error {
	block, _ := pem.Decode(certPEM)
	if block == nil || block.Type != "CERTIFICATE" {
		return errors.New("response is not a PEM certificate")
	}

	if err := os.WriteFile(CertPath(port), certPEM, 0644); err != nil {
		return fmt.Errorf("writing %s: %w", CertPath(port), err)
	}

	fmt.Println("Saved signed cert to", CertPath(port))
	return nil
}

func GenerateCACert() error {
	key, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return fmt.Errorf("generating CA key: %w", err)
	}

	serialNum, err := newSerialNumber()
	if err != nil {
		return fmt.Errorf("generating serial num: %w", err)
	}

	caTemplate := x509.Certificate{
		SerialNumber: serialNum,
		Subject: pkix.Name{
			Organization: []string{"CC"},
			CommonName:   "Go-TCP-Chat Mirror CA",
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(10 * 365 * 24 * time.Hour),

		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            0,
		MaxPathLenZero:        true,
	}

	der, err := x509.CreateCertificate(rand.Reader, &caTemplate, &caTemplate, key.Public(), key)
	if err != nil {
		return fmt.Errorf("creating CA cert: %w", err)
	}

	if err := writeKeyPair(der, key, CACertPath(), CAKeyPath()); err != nil {
		return err
	}

	fmt.Println("Generated", CACertPath(), "and", CAKeyPath())
	return nil
}

func LoadOrGenerateCA() (*x509.Certificate, *rsa.PrivateKey, error) {
	if _, err := os.Stat(CAKeyPath()); errors.Is(err, os.ErrNotExist) {
		if err := GenerateCACert(); err != nil {
			return nil, nil, err
		}
	}

	certPEM, err := os.ReadFile(CACertPath())
	if err != nil {
		return nil, nil, fmt.Errorf("reading CA cert: %w", err)
	}
	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil {
		return nil, nil, fmt.Errorf("no PEM data in %s", CACertPath())
	}
	caCert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing CA cert: %w", err)
	}

	keyPEM, err := os.ReadFile(CAKeyPath())
	if err != nil {
		return nil, nil, fmt.Errorf("reading CA key: %w", err)
	}
	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		return nil, nil, fmt.Errorf("no PEM data in %s", CAKeyPath())
	}
	caKey, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing CA key: %w", err)
	}

	return caCert, caKey, nil
}

func Fingerprint(cert *x509.Certificate) string {
	sum := sha256.Sum256(cert.Raw)
	return hex.EncodeToString(sum[:])
}

func newSerialNumber() (*big.Int, error) {
	return rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
}

func writeKeyPair(der []byte, key *rsa.PrivateKey, certPath, keyPath string) error {
	certFile, err := os.Create(certPath)
	if err != nil {
		return fmt.Errorf("creating %s: %w", certPath, err)
	}
	defer certFile.Close()

	if err := pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: der}); err != nil {
		return fmt.Errorf("writing %s: %w", certPath, err)
	}

	return writeKey(key, keyPath)
}

func writeKey(key *rsa.PrivateKey, keyPath string) error {
	keyFile, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("creating %s: %w", keyPath, err)
	}
	defer keyFile.Close()

	b := x509.MarshalPKCS1PrivateKey(key)
	if err := pem.Encode(keyFile, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: b}); err != nil {
		return fmt.Errorf("writing %s: %w", keyPath, err)
	}

	return nil
}

func CertPath(port string) string {
	return filepath.Join("..", "certs", fmt.Sprintf("cert-%s.pem", port))
}

func KeyPath(port string) string {
	return filepath.Join("..", "certs", fmt.Sprintf("key-%s.pem", port))
}

func CACertPath() string {
	return filepath.Join("..", "certs", "ca-cert.pem")
}

func CAKeyPath() string {
	return filepath.Join("..", "certs", "ca-key.pem")
}
