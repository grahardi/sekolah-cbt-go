package handlers

import (
	"net/http"

	"github.com/grahardi/sekolah-cbt-go/internal/httpx"
)

type jawabanSoalRequest struct {
	TextJawaban string `json:"text_jawaban"`
	Correct     bool   `json:"correct"`
	Urutan      int    `json:"urutan"`
}

func (h *Handlers) CreateJawabanSoal(w http.ResponseWriter, r *http.Request) {
	soalID := r.PathValue("id")

	var req jawabanSoalRequest
	if err := httpx.DecodeJSON(r, &req); err != nil || req.TextJawaban == "" {
		httpx.WriteError(w, http.StatusBadRequest, "text_jawaban wajib diisi")
		return
	}

	var id string
	err := h.Pool.QueryRow(r.Context(), `
		INSERT INTO jawaban_soals (soal_id, text_jawaban, correct, urutan)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, soalID, req.TextJawaban, req.Correct, req.Urutan).Scan(&id)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "gagal membuat opsi jawaban")
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, map[string]string{"id": id})
}

func (h *Handlers) UpdateJawabanSoal(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req jawabanSoalRequest
	if err := httpx.DecodeJSON(r, &req); err != nil || req.TextJawaban == "" {
		httpx.WriteError(w, http.StatusBadRequest, "text_jawaban wajib diisi")
		return
	}

	tag, err := h.Pool.Exec(r.Context(), `
		UPDATE jawaban_soals
		SET text_jawaban = $1, correct = $2, urutan = $3, updated_at = now()
		WHERE id = $4
	`, req.TextJawaban, req.Correct, req.Urutan, id)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "gagal mengubah opsi jawaban")
		return
	}
	if tag.RowsAffected() == 0 {
		httpx.WriteError(w, http.StatusNotFound, "opsi jawaban tidak ditemukan")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]bool{"updated": true})
}

func (h *Handlers) DeleteJawabanSoal(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	tag, err := h.Pool.Exec(r.Context(), `DELETE FROM jawaban_soals WHERE id = $1`, id)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "gagal menghapus opsi jawaban")
		return
	}
	if tag.RowsAffected() == 0 {
		httpx.WriteError(w, http.StatusNotFound, "opsi jawaban tidak ditemukan")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]bool{"deleted": true})
}
