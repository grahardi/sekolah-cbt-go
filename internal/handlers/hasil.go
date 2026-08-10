package handlers

import (
	"net/http"

	"github.com/grahardi/sekolah-cbt-go/internal/httpx"
)

type hasilItem struct {
	PesertaID   string  `json:"peserta_id"`
	NoUjian     string  `json:"no_ujian"`
	Nama        string  `json:"nama"`
	Kelas       *string `json:"kelas,omitempty"`
	JumlahBenar int     `json:"jumlah_benar"`
	JumlahSalah int     `json:"jumlah_salah"`
	Nilai       float64 `json:"nilai"`
}

// ListHasil returns every peserta's result across a jadwal's ujian(s), for
// the admin panel's recap/rekap nilai view.
func (h *Handlers) ListHasil(w http.ResponseWriter, r *http.Request) {
	jadwalID := r.PathValue("id")

	rows, err := h.Pool.Query(r.Context(), `
		SELECT p.id, p.no_ujian, p.nama, p.kelas, hu.jumlah_benar, hu.jumlah_salah, hu.nilai
		FROM hasil_ujians hu
		JOIN ujians u ON u.id = hu.ujian_id
		JOIN pesertas p ON p.id = hu.peserta_id
		WHERE u.jadwal_id = $1
		ORDER BY p.nama
	`, jadwalID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "gagal mengambil hasil")
		return
	}
	defer rows.Close()

	var items []hasilItem
	for rows.Next() {
		var it hasilItem
		if err := rows.Scan(&it.PesertaID, &it.NoUjian, &it.Nama, &it.Kelas,
			&it.JumlahBenar, &it.JumlahSalah, &it.Nilai); err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "gagal membaca hasil")
			return
		}
		items = append(items, it)
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{"hasil": items})
}
