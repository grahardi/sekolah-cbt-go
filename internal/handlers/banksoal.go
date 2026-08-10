package handlers

import (
	"net/http"

	"github.com/grahardi/sekolah-cbt-go/internal/httpx"
)

type banksoalItem struct {
	ID             string `json:"id"`
	MatpelID       string `json:"matpel_id"`
	Nama           string `json:"nama"`
	DurasiMenit    int    `json:"durasi_menit"`
	AcakSoal       bool   `json:"acak_soal"`
	AcakJawaban    bool   `json:"acak_jawaban"`
	TampilkanHasil bool   `json:"tampilkan_hasil"`
}

type banksoalRequest struct {
	MatpelID       string `json:"matpel_id"`
	Nama           string `json:"nama"`
	DurasiMenit    int    `json:"durasi_menit"`
	AcakSoal       bool   `json:"acak_soal"`
	AcakJawaban    bool   `json:"acak_jawaban"`
	TampilkanHasil bool   `json:"tampilkan_hasil"`
}

func (h *Handlers) ListBanksoal(w http.ResponseWriter, r *http.Request) {
	rows, err := h.Pool.Query(r.Context(), `
		SELECT id, matpel_id, nama, durasi_menit, acak_soal, acak_jawaban, tampilkan_hasil
		FROM banksoals ORDER BY nama
	`)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "gagal mengambil bank soal")
		return
	}
	defer rows.Close()

	var items []banksoalItem
	for rows.Next() {
		var it banksoalItem
		if err := rows.Scan(&it.ID, &it.MatpelID, &it.Nama, &it.DurasiMenit,
			&it.AcakSoal, &it.AcakJawaban, &it.TampilkanHasil); err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "gagal membaca bank soal")
			return
		}
		items = append(items, it)
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"banksoal": items})
}

func (h *Handlers) CreateBanksoal(w http.ResponseWriter, r *http.Request) {
	var req banksoalRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.MatpelID == "" || req.Nama == "" {
		httpx.WriteError(w, http.StatusBadRequest, "matpel_id dan nama wajib diisi")
		return
	}
	if req.DurasiMenit <= 0 {
		req.DurasiMenit = 90
	}

	var id string
	err := h.Pool.QueryRow(r.Context(), `
		INSERT INTO banksoals (matpel_id, nama, durasi_menit, acak_soal, acak_jawaban, tampilkan_hasil)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, req.MatpelID, req.Nama, req.DurasiMenit, req.AcakSoal, req.AcakJawaban, req.TampilkanHasil,
	).Scan(&id)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "gagal membuat bank soal")
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, banksoalItem{
		ID: id, MatpelID: req.MatpelID, Nama: req.Nama, DurasiMenit: req.DurasiMenit,
		AcakSoal: req.AcakSoal, AcakJawaban: req.AcakJawaban, TampilkanHasil: req.TampilkanHasil,
	})
}

func (h *Handlers) UpdateBanksoal(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req banksoalRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.MatpelID == "" || req.Nama == "" {
		httpx.WriteError(w, http.StatusBadRequest, "matpel_id dan nama wajib diisi")
		return
	}
	if req.DurasiMenit <= 0 {
		req.DurasiMenit = 90
	}

	tag, err := h.Pool.Exec(r.Context(), `
		UPDATE banksoals
		SET matpel_id = $1, nama = $2, durasi_menit = $3, acak_soal = $4,
		    acak_jawaban = $5, tampilkan_hasil = $6, updated_at = now()
		WHERE id = $7
	`, req.MatpelID, req.Nama, req.DurasiMenit, req.AcakSoal, req.AcakJawaban, req.TampilkanHasil, id)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "gagal mengubah bank soal")
		return
	}
	if tag.RowsAffected() == 0 {
		httpx.WriteError(w, http.StatusNotFound, "bank soal tidak ditemukan")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]bool{"updated": true})
}

func (h *Handlers) DeleteBanksoal(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	tag, err := h.Pool.Exec(r.Context(), `DELETE FROM banksoals WHERE id = $1`, id)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "gagal menghapus bank soal")
		return
	}
	if tag.RowsAffected() == 0 {
		httpx.WriteError(w, http.StatusNotFound, "bank soal tidak ditemukan")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]bool{"deleted": true})
}
