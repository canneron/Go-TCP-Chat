package model

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net"
	"strings"
)

type Node struct {
	Hostname   string          `json:"hostname"`
	Port       string          `json:"port"`
	Nickname   string          `json:"nickname"`
	Connection net.Conn        `json:"-"`
	Channel    Channel         `json:"channel"`
	ID         *Identification `json:"-"`
}

func (n Node) Address() string {
	return fmt.Sprintf("%s:%s", strings.TrimSpace(n.Hostname), strings.TrimSpace(n.Port))
}

func (n Node) ToJson() []byte {
	jsonData, err := json.Marshal(n)
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return nil
	}

	return jsonData
}

func (n Node) HashID() string {
	data := fmt.Sprintf("%s:%s:%s", strings.TrimSpace(n.Hostname), strings.TrimSpace(n.Port), strings.TrimSpace(n.Nickname))

	hash := sha256.New()
	hash.Write([]byte(data))
	hashBytes := hash.Sum(nil)

	return fmt.Sprintf("%x", hashBytes)
}

func (n Node) SignHash() ([]byte, error) {
	hashID := n.HashID()

	hashBytes := []byte(hashID)

	signature, err := rsa.SignPKCS1v15(rand.Reader, n.ID.PrivateKey, crypto.SHA256, hashBytes)
	if err != nil {
		return nil, fmt.Errorf("error signing hash: %v", err)
	}

	return signature, nil
}

func (n Node) VerifySignature(hashID string, signature []byte) (bool, error) {
	hashBytes := []byte(hashID)

	err := rsa.VerifyPKCS1v15(&n.ID.PrivateKey.PublicKey, crypto.SHA256, hashBytes, signature)
	if err != nil {
		return false, fmt.Errorf("verification failed: %v", err)
	}

	return true, nil
}
