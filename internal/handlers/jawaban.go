package handlers

import (
	"net/http"

	"github.com/jackc/pgx/v5"

	"github.com/grahardi/sekolah-cbt-go/internal/httpx"
)

type jawabRequest struct {
	SoalID        string  `json:"soal_id"`
	JawabanSoalID *string `json:"jawaban_soal_id"`
	JawabanText   *string `json:"jawaban_text"`
}

// SubmitJawaban saves (or overwrites) a peserta's answer to one soal.
// Multiple-choice/listening answers are graded immediately since the
// correct option is already known; free-text answers are stored ungraded
// for manual review later.
func (h *Handlers) SubmitJawaban(w http.ResponseWriter, r *http.Request) {
	claims := pesertaClaimsFrom(r)

	var req jawabRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.SoalID == "" {
		httpx.WriteError(w, http.StatusBadRequest, "soal_id wajib diisi")
		return
	}
	if req.JawabanSoalID == nil && req.JawabanText == nil {
		httpx.WriteError(w, http.StatusBadRequest, "jawaban_soal_id atau jawaban_text wajib diisi")
		return
	}

	ctx := r.Context()

	// make sure the soal actually belongs to this peserta's banksoal —
	// otherwise a peserta could smuggle answers into another ujian's soal
	var soalBanksoalID string
	err := h.Pool.QueryRow(ctx,
		`SELECT banksoal_id FROM soals WHERE id = $1 AND deleted_at IS NULL`,
		req.SoalID,
	).Scan(&soalBanksoalID)
	switch {
	case err == pgx.ErrNoRows:
		httpx.WriteError(w, http.StatusNotFound, "soal tidak ditemukan")
		return
	case err != nil:
		httpx.WriteError(w, http.StatusInternalServerError, "gagal memeriksa soal")
		return
	}
	if soalBanksoalID != claims.BanksoalID {
		httpx.WriteError(w, http.StatusForbidden, "soal bukan bagian dari ujian ini")
		return
	}

	var isCorrect *bool
	if req.JawabanSoalID != nil {
		var correct bool
		err := h.Pool.QueryRow(ctx,
			`SELECT correct FROM jawaban_soals WHERE id = $1 AND soal_id = $2`,
			*req.JawabanSoalID, req.SoalID,
		).Scan(&correct)
		switch {
		case err == pgx.ErrNoRows:
			httpx.WriteError(w, http.StatusBadRequest, "opsi jawaban tidak valid untuk soal ini")
			return
		case err != nil:
			httpx.WriteError(w, http.StatusInternalServerError, "gagal memeriksa jawaban")
			return
		}
		isCorrect = &correct
	}

	_, err = h.Pool.Exec(ctx, `
		INSERT INTO jawaban_pesertas (siswa_ujian_id, soal_id, jawaban_soal_id, jawaban_text, is_correct)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (siswa_ujian_id, soal_id)
		DO UPDATE SET jawaban_soal_id = EXCLUDED.jawaban_soal_id,
		              jawaban_text = EXCLUDED.jawaban_text,
		              is_correct = EXCLUDED.is_correct,
		              updated_at = now()
	`, claims.SiswaUjianID, req.SoalID, req.JawabanSoalID, req.JawabanText, isCorrect)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "gagal menyimpan jawaban")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]bool{"saved": true})
}
