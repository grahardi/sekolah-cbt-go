// Package auth handles the peserta-facing JWT: this server issues it after
// a peserta logs in with no_ujian + password + a valid session token, and
// verifies it on every /ujian/* request afterward. This is separate from
// the admin JWT that Laravel will issue and this server will only verify
// (that piece lands once the admin API exists) — same secret, but a
// different token shape, so the two can't be swapped for each other.
package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type PesertaClaims struct {
	SiswaUjianID string `json:"sua"`
	UjianID      string `json:"uid"`
	BanksoalID   string `json:"bid"`
	jwt.RegisteredClaims
}

func IssuePesertaToken(secret, pesertaID, siswaUjianID, ujianID, banksoalID string, expiresAt time.Time) (string, error) {
	claims := PesertaClaims{
		SiswaUjianID: siswaUjianID,
		UjianID:      ujianID,
		BanksoalID:   banksoalID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   pesertaID,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func ParsePesertaToken(secret, tokenStr string) (*PesertaClaims, error) {
	claims := &PesertaClaims{}
	_, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		return nil, err
	}
	return claims, nil
}
