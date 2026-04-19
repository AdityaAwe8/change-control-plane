package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/change-control-plane/change-control-plane/pkg/types"
)

var ErrInvalidToken = errors.New("invalid token")

type Claims struct {
	Version        string          `json:"version"`
	Subject        string          `json:"sub"`
	ActorType      types.ActorType `json:"actor_type"`
	AuthMethod     string          `json:"auth_method,omitempty"`
	AuthProviderID string          `json:"auth_provider_id,omitempty"`
	AuthProvider   string          `json:"auth_provider,omitempty"`
	IssuedAt       int64           `json:"iat"`
	ExpiresAt      int64           `json:"exp"`
}

type TokenService struct {
	secret []byte
	ttl    time.Duration
}

func NewTokenService(secret string, ttl time.Duration) *TokenService {
	return &TokenService{
		secret: []byte(secret),
		ttl:    ttl,
	}
}

func (s *TokenService) Sign(subject string, actorType types.ActorType) (string, error) {
	return s.SignDetailed(subject, actorType, "", "", "")
}

func (s *TokenService) SignDetailed(subject string, actorType types.ActorType, authMethod, authProviderID, authProvider string) (string, error) {
	claims := Claims{
		Version:        "v1",
		Subject:        subject,
		ActorType:      actorType,
		AuthMethod:     strings.TrimSpace(authMethod),
		AuthProviderID: strings.TrimSpace(authProviderID),
		AuthProvider:   strings.TrimSpace(authProvider),
		IssuedAt:       time.Now().UTC().Unix(),
		ExpiresAt:      time.Now().UTC().Add(s.ttl).Unix(),
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	signature := s.sign(encodedPayload)
	return fmt.Sprintf("ccp.%s.%s", encodedPayload, signature), nil
}

func (s *TokenService) Verify(token string) (Claims, error) {
	var claims Claims
	parts := strings.Split(token, ".")
	if len(parts) != 3 || parts[0] != "ccp" {
		return claims, ErrInvalidToken
	}
	encodedPayload := parts[1]
	if !hmac.Equal([]byte(parts[2]), []byte(s.sign(encodedPayload))) {
		return claims, ErrInvalidToken
	}
	payload, err := base64.RawURLEncoding.DecodeString(encodedPayload)
	if err != nil {
		return claims, ErrInvalidToken
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return claims, ErrInvalidToken
	}
	if time.Now().UTC().Unix() > claims.ExpiresAt {
		return claims, ErrInvalidToken
	}
	return claims, nil
}

func (s *TokenService) sign(encodedPayload string) string {
	mac := hmac.New(sha256.New, s.secret)
	_, _ = mac.Write([]byte(encodedPayload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (s *TokenService) GenerateAPIToken() (raw string, prefix string, hash string, err error) {
	secret := make([]byte, 24)
	if _, err = rand.Read(secret); err != nil {
		return "", "", "", err
	}
	prefixSeed := make([]byte, 6)
	if _, err = rand.Read(prefixSeed); err != nil {
		return "", "", "", err
	}
	prefix = "ccpt_" + hex.EncodeToString(prefixSeed)
	raw = prefix + "_" + hex.EncodeToString(secret)
	hash = s.HashOpaqueToken(raw)
	return raw, prefix, hash, nil
}

func (s *TokenService) GenerateBrowserSessionToken() (raw string, hash string, err error) {
	secret := make([]byte, 32)
	if _, err = rand.Read(secret); err != nil {
		return "", "", err
	}
	raw = "ccps_" + hex.EncodeToString(secret)
	hash = s.HashOpaqueToken(raw)
	return raw, hash, nil
}

func (s *TokenService) HashOpaqueToken(token string) string {
	digest := sha256.Sum256([]byte(token))
	return hex.EncodeToString(digest[:])
}
