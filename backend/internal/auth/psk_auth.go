package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// PSKAuthenticator 预共享密钥认证器
type PSKAuthenticator struct{}

// NewPSKAuthenticator 创建PSK认证器
func NewPSKAuthenticator() *PSKAuthenticator {
	return &PSKAuthenticator{}
}

// HashPSK 计算PSK的HMAC-SHA256哈希
func (a *PSKAuthenticator) HashPSK(psk string) string {
	h := hmac.New(sha256.New, []byte("edgelink-salt"))
	h.Write([]byte(psk))
	return hex.EncodeToString(h.Sum(nil))
}

// VerifyPSK 验证PSK哈希
func (a *PSKAuthenticator) VerifyPSK(psk, expectedHash string) bool {
	actualHash := a.HashPSK(psk)
	return hmac.Equal([]byte(actualHash), []byte(expectedHash))
}

// ValidatePSKFormat 验证PSK格式（32字符十六进制）
func (a *PSKAuthenticator) ValidatePSKFormat(psk string) error {
	if len(psk) != 64 {
		return fmt.Errorf("PSK must be 64 characters (32 bytes hex-encoded)")
	}
	if _, err := hex.DecodeString(psk); err != nil {
		return fmt.Errorf("PSK must be valid hexadecimal: %w", err)
	}
	return nil
}
