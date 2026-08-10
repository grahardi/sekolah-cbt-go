package handlers

import (
	"net/http"
	"time"

	"github.com/grahardi/sekolah-cbt-go/internal/httpx"
)

type jadwalItem struct {
	ID         string    `json:"id"`
	BanksoalID string    `json:"banksoal_id"`
	Nama       string    `json:"nama"`
	MulaiAt    time.Time `json:"mulai_at"`
	SelesaiAt  time.Time `json:"selesai_at"`
}

type jadwalRequest struct {
	Nama      string    `json:"nama"`
	MulaiAt   time.Time `json:"mulai_at"`
	SelesaiAt time.Time `json:"selesai_at"`
}

func (h *Handlers) ListJadwal(w http.ResponseWriter, r *http.Request) {
	banksoalID := r.PathValue("id")
	rows, err := h.Pool.Query(r.Context(), `
		SELECT id, banksoal_id, nama, mulai_at, selesai_at
		FROM jadwals WHERE banksoal_id = $1 ORDER BY mulai_at
	`, banksoalID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "gagal mengambil jadwal")
		return
	}
	defer rows.Close()

	var items []jadwalItem
	for rows.Next() {
		var it jadwalItem
		if err := rows.Scan(&it.ID, &it.BanksoalID, &it.Nama, &it.MulaiAt, &it.SelesaiAt); err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "gagal membaca jadwal")
			return
		}
		items = append(items, it)
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"jadwal": items})
}

func (h *Handlers) CreateJadwal(w http.ResponseWriter, r *http.Request) {
	banksoalID := r.PathValue("id")

	var req jadwalRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Nama == "" || req.MulaiAt.IsZero() || req.SelesaiAt.IsZero() {
		httpx.WriteError(w, http.StatusBadRequest, "nama, mulai_at, selesai_at wajib diisi")
		return
	}
	if !req.SelesaiAt.After(req.MulaiAt) {
		httpx.WriteError(w, http.StatusBadRequest, "selesai_at harus setelah mulai_at")
		return
	}

	var id string
	err := h.Pool.QueryRow(r.Context(), `
		INSERT INTO jadwals (banksoal_id, nama, mulai_at, selesai_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, banksoalID, req.Nama, req.MulaiAt, req.SelesaiAt).Scan(&id)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "gagal membuat jadwal")
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, map[string]string{"id": id})
}

func (h *Handlers) UpdateJadwal(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req jadwalRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Nama == "" || req.MulaiAt.IsZero() || req.SelesaiAt.IsZero() {
		httpx.WriteError(w, http.StatusBadRequest, "nama, mulai_at, selesai_at wajib diisi")
		return
	}
	if !req.SelesaiAt.After(req.MulaiAt) {
		httpx.WriteError(w, http.StatusBadRequest, "selesai_at harus setelah mulai_at")
		return
	}

	tag, err := h.Pool.Exec(r.Context(), `
		UPDATE jadwals SET nama = $1, mulai_at = $2, selesai_at = $3, updated_at = now()
		WHERE id = $4
	`, req.Nama, req.MulaiAt, req.SelesaiAt, id)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "gagal mengubah jadwal")
		return
	}
	if tag.RowsAffected() == 0 {
		httpx.WriteError(w, http.StatusNotFound, "jadwal tidak ditemukan")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]bool{"updated": true})
}

func (h *Handlers) DeleteJadwal(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	tag, err := h.Pool.Exec(r.Context(), `DELETE FROM jadwals WHERE id = $1`, id)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "gagal menghapus jadwal")
		return
	}
	if tag.RowsAffected() == 0 {
		httpx.WriteError(w, http.StatusNotFound, "jadwal tidak ditemukan")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]bool{"deleted": true})
}
