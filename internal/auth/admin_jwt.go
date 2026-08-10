package auth

import (
	"errors"

	"github.com/golang-jwt/jwt/v5"
)

// AdminClaims is what Laravel issues once an admin/guru opens the CBT
// section for their sekolah. This server only ever verifies these — it
// never signs one itself — using the same shared JWT_SECRET as this
// instance's peserta tokens. Laravel must include "typ": "admin" and
// "sekolah_id" matching this instance's SEKOLAH_ID in every token it
// issues, or RequireAdmin will reject it.
type AdminClaims struct {
	Typ       string `json:"typ"`
	Role      string `json:"role"`
	SekolahID string `json:"sekolah_id"`
	jwt.RegisteredClaims
}

func ParseAdminToken(secret, tokenStr string) (*AdminClaims, error) {
	claims := &AdminClaims{}
	_, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		return nil, err
	}
	if claims.Typ != "admin" {
		return nil, errors.New("wrong token type")
	}
	return claims, nil
}
