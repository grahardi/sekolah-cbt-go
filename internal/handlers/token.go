package handlers

import (
	"crypto/rand"
	"net/http"

	"github.com/grahardi/sekolah-cbt-go/internal/httpx"
)

// no O/0/I/1 — avoids characters proctors or peserta could misread/mistype
const tokenAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func generateToken(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	for i := range b {
		b[i] = tokenAlphabet[int(b[i])%len(tokenAlphabet)]
	}
	return string(b), nil
}

type tokenItem struct {
	ID     string `json:"id"`
	Token  string `json:"token"`
	Status string `json:"status"`
}

// CreateToken generates a fresh session token for a jadwal, inactive by
// default — the proctor activates it from the panel right before the exam
// starts, same flow as tokens in the legacy CBT.
func (h *Handlers) CreateToken(w http.ResponseWriter, r *http.Request) {
	jadwalID := r.PathValue("id")

	tokenStr, err := generateToken(6)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "gagal membuat token")
		return
	}

	var id string
	err = h.Pool.QueryRow(r.Context(), `
		INSERT INTO tokens (jadwal_id, token, status)
		VALUES ($1, $2, 'nonaktif')
		RETURNING id
	`, jadwalID, tokenStr).Scan(&id)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "gagal menyimpan token")
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, tokenItem{ID: id, Token: tokenStr, Status: "nonaktif"})
}

type tokenStatusRequest struct {
	Status string `json:"status"`
}

// SetTokenStatus toggles a token aktif/nonaktif — the on/off switch
// proctors use to open and close peserta login for a session.
func (h *Handlers) SetTokenStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req tokenStatusRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Status != "aktif" && req.Status != "nonaktif" {
		httpx.WriteError(w, http.StatusBadRequest, "status harus 'aktif' atau 'nonaktif'")
		return
	}

	tag, err := h.Pool.Exec(r.Context(),
		`UPDATE tokens SET status = $1, updated_at = now() WHERE id = $2`, req.Status, id)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "gagal mengubah status token")
		return
	}
	if tag.RowsAffected() == 0 {
		httpx.WriteError(w, http.StatusNotFound, "token tidak ditemukan")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]bool{"updated": true})
}
