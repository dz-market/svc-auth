package security

import (
	"crypto/rsa"
	"time"
)

type TokenManager struct {
	Access  AccessManager
	Refresh RefreshManager
}

func NewTokenManager(publicKey *rsa.PublicKey, privateKey *rsa.PrivateKey, accessTTL time.Duration, refreshByteLen int) TokenManager {
	return TokenManager{
		Access: AccessManager{
			publicKey:  publicKey,
			privateKey: privateKey,
			ttl:        accessTTL,
		},
		Refresh: RefreshManager{
			byteLen: refreshByteLen,
		},
	}
}
