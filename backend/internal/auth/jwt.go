package auth

import (
	"errors"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID int64    `json:"user_id"`
	Email  string   `json:"email"`
	Roles  []string `json:"roles"`
	jwt.RegisteredClaims
}

type TokenManager struct {
	secret        []byte
	accessExpires time.Duration
}

func NewTokenManager(secret string, accessExpires time.Duration) *TokenManager {
	if accessExpires <= 0 {
		accessExpires = time.Hour
	}

	return &TokenManager{
		secret:        []byte(secret),
		accessExpires: accessExpires,
	}
}

func (m *TokenManager) GenerateAccessToken(userID int64, email string, roles []string) (string, time.Time, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(m.accessExpires)

	claims := Claims{
		UserID: userID,
		Email:  email,
		Roles:  append([]string(nil), roles...),
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatInt(userID, 10),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, err
	}

	return signed, expiresAt, nil
}

func (m *TokenManager) ParseAccessToken(token string) (*Claims, error) {
	claims := &Claims{}
	parsed, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (any, error) {
		return m.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return nil, err
	}

	if !parsed.Valid {
		return nil, errors.New("token is invalid")
	}

	return claims, nil
}
