package handlers

import (
	"net/http"

	"github.com/grahardi/sekolah-cbt-go/internal/httpx"
)

type soalAdminOption struct {
	ID      string `json:"id"`
	Teks    string `json:"teks"`
	Correct bool   `json:"correct"`
	Urutan  int    `json:"urutan"`
}

type soalAdminItem struct {
	ID         string            `json:"id"`
	BanksoalID string            `json:"banksoal_id"`
	TipeSoal   int16             `json:"tipe_soal"`
	Pertanyaan string            `json:"pertanyaan"`
	Rujukan    *string           `json:"rujukan,omitempty"`
	Audio      *string           `json:"audio,omitempty"`
	Urutan     int               `json:"urutan"`
	Poin       float64           `json:"poin"`
	Opsi       []soalAdminOption `json:"opsi,omitempty"`
}

type soalRequest struct {
	TipeSoal   int16   `json:"tipe_soal"`
	Pertanyaan string  `json:"pertanyaan"`
	Rujukan    string  `json:"rujukan"`
	Audio      string  `json:"audio"`
	Urutan     int     `json:"urutan"`
	Poin       float64 `json:"poin"`
}

// ListSoalAdmin returns soal for a banksoal including the answer key —
// unlike the peserta-facing ListSoal in soal.go, which never sends `correct`.
func (h *Handlers) ListSoalAdmin(w http.ResponseWriter, r *http.Request) {
	banksoalID := r.PathValue("id")
	ctx := r.Context()

	rows, err := h.Pool.Query(ctx, `
		SELECT id, banksoal_id, tipe_soal, pertanyaan, rujukan, audio, urutan, poin
		FROM soals
		WHERE banksoal_id = $1 AND deleted_at IS NULL
		ORDER BY urutan
	`, banksoalID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "gagal mengambil soal")
		return
	}
	defer rows.Close()

	var items []soalAdminItem
	for rows.Next() {
		var it soalAdminItem
		if err := rows.Scan(&it.ID, &it.BanksoalID, &it.TipeSoal, &it.Pertanyaan,
			&it.Rujukan, &it.Audio, &it.Urutan, &it.Poin); err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "gagal membaca soal")
			return
		}
		items = append(items, it)
	}

	optRows, err := h.Pool.Query(ctx, `
		SELECT js.id, js.soal_id, js.text_jawaban, js.correct, js.urutan
		FROM jawaban_soals js
		JOIN soals s ON s.id = js.soal_id
		WHERE s.banksoal_id = $1 AND s.deleted_at IS NULL
		ORDER BY js.urutan
	`, banksoalID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "gagal mengambil opsi jawaban")
		return
	}
	defer optRows.Close()

	optionsBySoal := map[string][]soalAdminOption{}
	for optRows.Next() {
		var soalID string
		var opt soalAdminOption
		if err := optRows.Scan(&opt.ID, &soalID, &opt.Teks, &opt.Correct, &opt.Urutan); err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "gagal membaca opsi jawaban")
			return
		}
		optionsBySoal[soalID] = append(optionsBySoal[soalID], opt)
	}

	for i := range items {
		items[i].Opsi = optionsBySoal[items[i].ID]
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{"soal": items})
}

func (h *Handlers) CreateSoal(w http.ResponseWriter, r *http.Request) {
	banksoalID := r.PathValue("id")

	var req soalRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Pertanyaan == "" {
		httpx.WriteError(w, http.StatusBadRequest, "pertanyaan wajib diisi")
		return
	}
	if req.TipeSoal == 0 {
		req.TipeSoal = 1
	}
	if req.Poin == 0 {
		req.Poin = 1
	}

	var id string
	err := h.Pool.QueryRow(r.Context(), `
		INSERT INTO soals (banksoal_id, tipe_soal, pertanyaan, rujukan, audio, urutan, poin)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, banksoalID, req.TipeSoal, req.Pertanyaan, httpx.NullIfEmpty(req.Rujukan),
		httpx.NullIfEmpty(req.Audio), req.Urutan, req.Poin,
	).Scan(&id)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "gagal membuat soal")
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, map[string]string{"id": id})
}

func (h *Handlers) UpdateSoal(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req soalRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Pertanyaan == "" {
		httpx.WriteError(w, http.StatusBadRequest, "pertanyaan wajib diisi")
		return
	}

	tag, err := h.Pool.Exec(r.Context(), `
		UPDATE soals
		SET tipe_soal = $1, pertanyaan = $2, rujukan = $3, audio = $4,
		    urutan = $5, poin = $6, updated_at = now()
		WHERE id = $7 AND deleted_at IS NULL
	`, req.TipeSoal, req.Pertanyaan, httpx.NullIfEmpty(req.Rujukan),
		httpx.NullIfEmpty(req.Audio), req.Urutan, req.Poin, id)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "gagal mengubah soal")
		return
	}
	if tag.RowsAffected() == 0 {
		httpx.WriteError(w, http.StatusNotFound, "soal tidak ditemukan")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]bool{"updated": true})
}

// DeleteSoal soft-deletes so past jawaban_pesertas / hasil_ujians rows
// referencing this soal stay intact for historical results.
func (h *Handlers) DeleteSoal(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	tag, err := h.Pool.Exec(r.Context(),
		`UPDATE soals SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "gagal menghapus soal")
		return
	}
	if tag.RowsAffected() == 0 {
		httpx.WriteError(w, http.StatusNotFound, "soal tidak ditemukan")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]bool{"deleted": true})
}
