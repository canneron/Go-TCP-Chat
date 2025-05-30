package model

import (
	"crypto/rsa"
	"crypto/tls"
)

type Identification struct {
	PrivateKey  *rsa.PrivateKey
	Certificate *tls.Certificate
	Config      *tls.Config
}
