package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// KeyPair Ed25519密钥对
type KeyPair struct {
	PublicKey  ed25519.PublicKey
	PrivateKey ed25519.PrivateKey
}

// KeyPairBase64 Base64编码的密钥对
type KeyPairBase64 struct {
	PublicKey  string `json:"public_key"`
	PrivateKey string `json:"private_key"`
}

// GenerateKeyPair 生成Ed25519密钥对
func GenerateKeyPair() (*KeyPair, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %w", err)
	}

	return &KeyPair{
		PublicKey:  publicKey,
		PrivateKey: privateKey,
	}, nil
}

// ToBase64 将密钥对编码为Base64
func (kp *KeyPair) ToBase64() *KeyPairBase64 {
	return &KeyPairBase64{
		PublicKey:  base64.StdEncoding.EncodeToString(kp.PublicKey),
		PrivateKey: base64.StdEncoding.EncodeToString(kp.PrivateKey),
	}
}

// Sign 使用私钥对消息进行签名
func (kp *KeyPair) Sign(message []byte) []byte {
	return ed25519.Sign(kp.PrivateKey, message)
}

// Verify 验证签名
func (kp *KeyPair) Verify(message, signature []byte) bool {
	return ed25519.Verify(kp.PublicKey, message, signature)
}

// ParseKeyPairBase64 从Base64字符串解析密钥对
func ParseKeyPairBase64(pub, priv string) (*KeyPair, error) {
	publicKey, err := base64.StdEncoding.DecodeString(pub)
	if err != nil {
		return nil, fmt.Errorf("invalid public key: %w", err)
	}

	privateKey, err := base64.StdEncoding.DecodeString(priv)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}

	if len(publicKey) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid public key size: expected %d, got %d", ed25519.PublicKeySize, len(publicKey))
	}

	if len(privateKey) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("invalid private key size: expected %d, got %d", ed25519.PrivateKeySize, len(privateKey))
	}

	return &KeyPair{
		PublicKey:  ed25519.PublicKey(publicKey),
		PrivateKey: ed25519.PrivateKey(privateKey),
	}, nil
}

// PublicKeyFromBase64 从Base64字符串解析公钥
func PublicKeyFromBase64(pub string) (ed25519.PublicKey, error) {
	publicKey, err := base64.StdEncoding.DecodeString(pub)
	if err != nil {
		return nil, fmt.Errorf("invalid public key: %w", err)
	}

	if len(publicKey) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid public key size: expected %d, got %d", ed25519.PublicKeySize, len(publicKey))
	}

	return ed25519.PublicKey(publicKey), nil
}
