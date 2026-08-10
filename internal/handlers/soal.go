package handlers

import (
	"net/http"

	"github.com/grahardi/sekolah-cbt-go/internal/httpx"
)

type soalOption struct {
	ID     string `json:"id"`
	Teks   string `json:"teks"`
	Urutan int    `json:"urutan"`
}

type soalItem struct {
	ID            string       `json:"id"`
	TipeSoal      int16        `json:"tipe_soal"`
	Pertanyaan    string       `json:"pertanyaan"`
	Rujukan       *string      `json:"rujukan,omitempty"`
	Audio         *string      `json:"audio,omitempty"`
	Urutan        int          `json:"urutan"`
	Opsi          []soalOption `json:"opsi,omitempty"`
	JawabanSoalID *string      `json:"jawaban_soal_id,omitempty"`
	JawabanText   *string      `json:"jawaban_text,omitempty"`
}

// ListSoal returns every soal in the peserta's banksoal, each with its
// answer options (the `correct` flag is never sent to the client) and
// whatever this peserta has already answered so far, so the client can
// restore selections after a reload or reconnect.
//
// NOTE: acak_soal/acak_jawaban randomization isn't implemented yet — soal
// and opsi both come back in `urutan` order for every peserta. Worth adding
// as a per-siswa_ujian seeded shuffle once the engine is otherwise proven.
func (h *Handlers) ListSoal(w http.ResponseWriter, r *http.Request) {
	claims := pesertaClaimsFrom(r)
	ctx := r.Context()

	rows, err := h.Pool.Query(ctx, `
		SELECT s.id, s.tipe_soal, s.pertanyaan, s.rujukan, s.audio, s.urutan,
		       jp.jawaban_soal_id, jp.jawaban_text
		FROM soals s
		LEFT JOIN jawaban_pesertas jp
		       ON jp.soal_id = s.id AND jp.siswa_ujian_id = $2
		WHERE s.banksoal_id = $1 AND s.deleted_at IS NULL
		ORDER BY s.urutan
	`, claims.BanksoalID, claims.SiswaUjianID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "gagal mengambil soal")
		return
	}
	defer rows.Close()

	var items []soalItem
	for rows.Next() {
		var it soalItem
		if err := rows.Scan(&it.ID, &it.TipeSoal, &it.Pertanyaan, &it.Rujukan, &it.Audio, &it.Urutan,
			&it.JawabanSoalID, &it.JawabanText); err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "gagal membaca soal")
			return
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "gagal membaca soal")
		return
	}

	optRows, err := h.Pool.Query(ctx, `
		SELECT js.id, js.soal_id, js.text_jawaban, js.urutan
		FROM jawaban_soals js
		JOIN soals s ON s.id = js.soal_id
		WHERE s.banksoal_id = $1 AND s.deleted_at IS NULL
		ORDER BY js.urutan
	`, claims.BanksoalID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "gagal mengambil opsi jawaban")
		return
	}
	defer optRows.Close()

	optionsBySoal := map[string][]soalOption{}
	for optRows.Next() {
		var soalID string
		var opt soalOption
		if err := optRows.Scan(&opt.ID, &soalID, &opt.Teks, &opt.Urutan); err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "gagal membaca opsi jawaban")
			return
		}
		optionsBySoal[soalID] = append(optionsBySoal[soalID], opt)
	}
	if err := optRows.Err(); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "gagal membaca opsi jawaban")
		return
	}

	for i := range items {
		items[i].Opsi = optionsBySoal[items[i].ID]
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{"soal": items})
}
