package certs

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

func GenerateCert(host, port string) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	address := fmt.Sprintf("%s:%s", host, port)
	serialNum, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		fmt.Println("Error generating serial num: ", err)
		return
	}

	tempCert := x509.Certificate{
		SerialNumber: serialNum,
		Subject: pkix.Name{
			Organization: []string{"CC"},
			CommonName:   address,
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(365 * 24 * time.Hour),

		KeyUsage: x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
			x509.ExtKeyUsageClientAuth,
		},
		BasicConstraintsValid: true,
		DNSNames:              []string{address},
	}

	der, err := x509.CreateCertificate(rand.Reader, &tempCert, &tempCert, key.Public(), key)

	if err != nil {
		fmt.Println("Error generating cert: ", err)
	}

	certsDir := filepath.Join("..", "certs")

	certPath := filepath.Join(certsDir, "cert.pem")
	keyPath := filepath.Join(certsDir, "key.pem")
	fmt.Println("Path: ", certPath)

	certFile, err1 := os.Create(certPath)
	keyFile, _ := os.Create(keyPath)

	if err1 != nil {
		fmt.Println("error,", err1)
	}

	pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: der})
	certFile.Close()

	b := x509.MarshalPKCS1PrivateKey(key)
	pem.Encode(keyFile, &pem.Block{Type: "PRIVATE KEY", Bytes: b})
	keyFile.Close()

	fmt.Println("Generated cert.pem and key.pem")
}
