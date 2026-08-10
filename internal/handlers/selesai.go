package handlers

import (
	"net/http"

	"github.com/grahardi/sekolah-cbt-go/internal/httpx"
)

type selesaiResponse struct {
	JumlahBenar int     `json:"jumlah_benar"`
	JumlahSalah int     `json:"jumlah_salah"`
	Nilai       float64 `json:"nilai"`
}

// SelesaiUjian closes out the peserta's attempt and scores every
// multiple-choice/listening soal (tipe_soal 1 and 3, both graded via a
// jawaban_soal_id option). Essay/short-answer soal are left ungraded here
// for manual review later.
func (h *Handlers) SelesaiUjian(w http.ResponseWriter, r *http.Request) {
	claims := pesertaClaimsFrom(r)
	ctx := r.Context()

	tag, err := h.Pool.Exec(ctx, `
		UPDATE siswa_ujians
		SET status = 'selesai', waktu_selesai = now(), updated_at = now()
		WHERE id = $1 AND status = 'berlangsung'
	`, claims.SiswaUjianID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "gagal menyelesaikan ujian")
		return
	}
	if tag.RowsAffected() == 0 {
		httpx.WriteError(w, http.StatusConflict, "ujian sudah diselesaikan sebelumnya")
		return
	}

	var benar, salah int
	err = h.Pool.QueryRow(ctx, `
		SELECT
			count(*) FILTER (WHERE jp.is_correct = true),
			count(*) FILTER (WHERE jp.is_correct = false)
		FROM jawaban_pesertas jp
		WHERE jp.siswa_ujian_id = $1
	`, claims.SiswaUjianID).Scan(&benar, &salah)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "gagal menghitung hasil")
		return
	}

	var totalGradable int
	err = h.Pool.QueryRow(ctx, `
		SELECT count(*) FROM soals
		WHERE banksoal_id = $1 AND deleted_at IS NULL AND tipe_soal IN (1, 3)
	`, claims.BanksoalID).Scan(&totalGradable)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "gagal menghitung total soal")
		return
	}

	var nilai float64
	if totalGradable > 0 {
		nilai = float64(benar) / float64(totalGradable) * 100
	}

	_, err = h.Pool.Exec(ctx, `
		INSERT INTO hasil_ujians (ujian_id, peserta_id, jumlah_benar, jumlah_salah, nilai)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (ujian_id, peserta_id)
		DO UPDATE SET jumlah_benar = EXCLUDED.jumlah_benar,
		              jumlah_salah = EXCLUDED.jumlah_salah,
		              nilai = EXCLUDED.nilai,
		              updated_at = now()
	`, claims.UjianID, claims.Subject, benar, salah, nilai)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "gagal menyimpan hasil")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, selesaiResponse{
		JumlahBenar: benar,
		JumlahSalah: salah,
		Nilai:       nilai,
	})
}
