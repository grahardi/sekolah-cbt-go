// Package handlers holds the peserta-facing exam engine: login, fetching
// soal, submitting jawaban, and finishing the attempt. Admin endpoints
// (CRUD soal/banksoal/jadwal) land here later once the Laravel<->Go SSO
// piece exists — deliberately not built yet so this stage stays testable
// end-to-end with just psql for seeding.
package handlers

import "github.com/jackc/pgx/v5/pgxpool"

type Handlers struct {
	Pool      *pgxpool.Pool
	JWTSecret string
}
