package handlers

import (
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/grahardi/sekolah-cbt-go/internal/auth"
	"github.com/grahardi/sekolah-cbt-go/internal/httpx"
)

type loginRequest struct {
	NoUjian  string `json:"no_ujian"`
	Password string `json:"password"`
	Token    string `json:"token"`
}

type loginResponse struct {
	AccessToken string    `json:"access_token"`
	ExpiresAt   time.Time `json:"expires_at"`
	Nama        string    `json:"nama"`
}

// PesertaLogin authenticates a student with their own no_ujian+password
// plus the session token the proctor activated for this jadwal, opens (or
// resumes) their siswa_ujian attempt, and hands back a scoped JWT good
// until the sooner of durasi_menit-from-now or the jadwal's selesai_at.
func (h *Handlers) PesertaLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.NoUjian == "" || req.Password == "" || req.Token == "" {
		httpx.WriteError(w, http.StatusBadRequest, "no_ujian, password, dan token wajib diisi")
		return
	}

	ctx := r.Context()

	var pesertaID, passwordHash, nama string
	var status int16
	err := h.Pool.QueryRow(ctx,
		`SELECT id, password_hash, nama, status FROM pesertas WHERE no_ujian = $1`,
		req.NoUjian,
	).Scan(&pesertaID, &passwordHash, &nama, &status)
	switch {
	case err == pgx.ErrNoRows:
		httpx.WriteError(w, http.StatusUnauthorized, "no ujian atau password salah")
		return
	case err != nil:
		httpx.WriteError(w, http.StatusInternalServerError, "gagal memeriksa peserta")
		return
	}
	if status != 1 {
		httpx.WriteError(w, http.StatusForbidden, "akun peserta tidak aktif")
		return
	}
	if !auth.CheckPassword(passwordHash, req.Password) {
		httpx.WriteError(w, http.StatusUnauthorized, "no ujian atau password salah")
		return
	}

	var jadwalID, banksoalID string
	var durasiMenit int
	var selesaiAt time.Time
	err = h.Pool.QueryRow(ctx, `
		SELECT j.id, j.banksoal_id, b.durasi_menit, j.selesai_at
		FROM tokens t
		JOIN jadwals j ON j.id = t.jadwal_id
		JOIN banksoals b ON b.id = j.banksoal_id
		WHERE t.token = $1 AND t.status = 'aktif'
	`, req.Token).Scan(&jadwalID, &banksoalID, &durasiMenit, &selesaiAt)
	switch {
	case err == pgx.ErrNoRows:
		httpx.WriteError(w, http.StatusUnauthorized, "token ujian tidak valid atau belum diaktifkan")
		return
	case err != nil:
		httpx.WriteError(w, http.StatusInternalServerError, "gagal memeriksa token")
		return
	}

	var ujianID string
	err = h.Pool.QueryRow(ctx,
		`SELECT id FROM ujians WHERE jadwal_id = $1 AND status = 'aktif'`,
		jadwalID,
	).Scan(&ujianID)
	switch {
	case err == pgx.ErrNoRows:
		httpx.WriteError(w, http.StatusConflict, "ujian belum dimulai oleh panitia")
		return
	case err != nil:
		httpx.WriteError(w, http.StatusInternalServerError, "gagal memeriksa ujian")
		return
	}

	var siswaUjianID string
	var waktuMulai time.Time
	err = h.Pool.QueryRow(ctx, `
		INSERT INTO siswa_ujians (ujian_id, peserta_id, waktu_mulai, status)
		VALUES ($1, $2, now(), 'berlangsung')
		ON CONFLICT (ujian_id, peserta_id)
		DO UPDATE SET updated_at = now()
		RETURNING id, waktu_mulai
	`, ujianID, pesertaID).Scan(&siswaUjianID, &waktuMulai)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "gagal memulai sesi ujian")
		return
	}

	deadline := waktuMulai.Add(time.Duration(durasiMenit) * time.Minute)
	if selesaiAt.Before(deadline) {
		deadline = selesaiAt
	}

	accessToken, err := auth.IssuePesertaToken(h.JWTSecret, pesertaID, siswaUjianID, ujianID, banksoalID, deadline)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "gagal membuat token akses")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, loginResponse{
		AccessToken: accessToken,
		ExpiresAt:   deadline,
		Nama:        nama,
	})
}
