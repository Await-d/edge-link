package auth

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"time"
)

// DeviceSignatureVerifier 设备签名验证器
type DeviceSignatureVerifier struct {
	maxClockSkew time.Duration // 允许的时钟偏差
}

// NewDeviceSignatureVerifier 创建设备签名验证器
func NewDeviceSignatureVerifier(maxClockSkew time.Duration) *DeviceSignatureVerifier {
	return &DeviceSignatureVerifier{
		maxClockSkew: maxClockSkew,
	}
}

// VerifySignature 验证Ed25519签名
// message: 要验证的消息（通常是请求体 + 时间戳）
// signatureB64: Base64编码的签名
// publicKeyB64: Base64编码的公钥
func (v *DeviceSignatureVerifier) VerifySignature(message []byte, signatureB64, publicKeyB64 string) error {
	// 解码签名
	signature, err := base64.StdEncoding.DecodeString(signatureB64)
	if err != nil {
		return fmt.Errorf("invalid signature encoding: %w", err)
	}

	if len(signature) != ed25519.SignatureSize {
		return fmt.Errorf("invalid signature length: expected %d, got %d", ed25519.SignatureSize, len(signature))
	}

	// 解码公钥
	publicKey, err := base64.StdEncoding.DecodeString(publicKeyB64)
	if err != nil {
		return fmt.Errorf("invalid public key encoding: %w", err)
	}

	if len(publicKey) != ed25519.PublicKeySize {
		return fmt.Errorf("invalid public key length: expected %d, got %d", ed25519.PublicKeySize, len(publicKey))
	}

	// 验证签名
	if !ed25519.Verify(ed25519.PublicKey(publicKey), message, signature) {
		return fmt.Errorf("signature verification failed")
	}

	return nil
}

// VerifyTimestamp 验证时间戳（防止重放攻击）
func (v *DeviceSignatureVerifier) VerifyTimestamp(timestamp time.Time) error {
	now := time.Now()
	diff := now.Sub(timestamp)

	if diff < 0 {
		diff = -diff
	}

	if diff > v.maxClockSkew {
		return fmt.Errorf("timestamp outside acceptable range (±%v): %v", v.maxClockSkew, diff)
	}

	return nil
}

// ConstructMessage 构建待签名消息（请求体 + 时间戳）
func (v *DeviceSignatureVerifier) ConstructMessage(body []byte, timestamp string) []byte {
	return append(body, []byte(timestamp)...)
}
