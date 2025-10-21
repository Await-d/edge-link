package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Claims JWT声明
type Claims struct {
	UserID         uuid.UUID `json:"user_id"`
	Email          string    `json:"email"`
	Role           string    `json:"role"`
	OrganizationID uuid.UUID `json:"organization_id"`
	jwt.RegisteredClaims
}

// JWTManager JWT令牌管理器
type JWTManager struct {
	secretKey     []byte
	tokenDuration time.Duration
}

// NewJWTManager 创建JWT管理器
func NewJWTManager(secretKey string, tokenDuration time.Duration) *JWTManager {
	return &JWTManager{
		secretKey:     []byte(secretKey),
		tokenDuration: tokenDuration,
	}
}

// GenerateToken 生成JWT令牌
func (m *JWTManager) GenerateToken(userID, orgID uuid.UUID, email, role string) (string, error) {
	claims := &Claims{
		UserID:         userID,
		Email:          email,
		Role:           role,
		OrganizationID: orgID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.tokenDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "edgelink-control-plane",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secretKey)
}

// ValidateToken 验证JWT令牌
func (m *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// 验证签名方法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.secretKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

// RefreshToken 刷新JWT令牌
func (m *JWTManager) RefreshToken(tokenString string) (string, error) {
	claims, err := m.ValidateToken(tokenString)
	if err != nil {
		return "", err
	}

	// 检查令牌是否即将过期（在剩余时间少于1小时时允许刷新）
	if time.Until(claims.ExpiresAt.Time) > time.Hour {
		return "", fmt.Errorf("token not eligible for refresh yet")
	}

	// 生成新令牌
	return m.GenerateToken(claims.UserID, claims.OrganizationID, claims.Email, claims.Role)
}

// ExtractTokenFromHeader 从Authorization头提取令牌
func (m *JWTManager) ExtractTokenFromHeader(authHeader string) (string, error) {
	const bearerPrefix = "Bearer "
	if len(authHeader) < len(bearerPrefix) {
		return "", fmt.Errorf("invalid authorization header format")
	}

	if authHeader[:len(bearerPrefix)] != bearerPrefix {
		return "", fmt.Errorf("authorization header must start with 'Bearer '")
	}

	return authHeader[len(bearerPrefix):], nil
}

// TODO: 集成OIDC/SAML
// 以下方法为占位符，待集成真实的OIDC/SAML提供商

// ValidateOIDCToken OIDC令牌验证（占位符）
func (m *JWTManager) ValidateOIDCToken(oidcToken string) (*Claims, error) {
	// TODO: 实现OIDC令牌验证
	// 1. 从OIDC提供商获取公钥
	// 2. 验证令牌签名
	// 3. 提取用户信息
	// 4. 映射到内部Claims结构
	return nil, fmt.Errorf("OIDC integration not implemented yet")
}

// ValidateSAMLAssertion SAML断言验证（占位符）
func (m *JWTManager) ValidateSAMLAssertion(samlAssertion string) (*Claims, error) {
	// TODO: 实现SAML断言验证
	// 1. 解析SAML XML
	// 2. 验证签名
	// 3. 提取用户属性
	// 4. 映射到内部Claims结构
	return nil, fmt.Errorf("SAML integration not implemented yet")
}
