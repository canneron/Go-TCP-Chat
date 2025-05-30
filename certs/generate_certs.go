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
	"time"
)

func GenerateCert(nickname, host, port string) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	id := fmt.Sprintf("%s:%s:%s", host, port, nickname)
	serialNum, _ := rand.Int(rand.Reader, new(big.Int))

	tempCert := x509.Certificate{
		SerialNumber: serialNum,
		Subject: pkix.Name{
			Organization: []string{"CC"},
			CommonName:   id,
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(365 * 24 * time.Hour),

		KeyUsage: x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
			x509.ExtKeyUsageClientAuth,
		},
		BasicConstraintsValid: true,
		DNSNames:              []string{id},
	}

	der, err := x509.CreateCertificate(rand.Reader, &tempCert, &tempCert, key.Public(), key)

	if err != nil {
		fmt.Println("Error generating cert: ", err)
	}

	certFile, _ := os.Create("cert.pem")
	keyFile, _ := os.Create("key.pem")

	pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: der})
	certFile.Close()

	b := x509.MarshalPKCS1PrivateKey(key)
	pem.Encode(keyFile, &pem.Block{Type: "PRIVATE KEY", Bytes: b})
	keyFile.Close()

	fmt.Println("Generated cert.pem and key.pem")
}
