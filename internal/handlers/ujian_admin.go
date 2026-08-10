package handlers

import (
	"net/http"

	"github.com/jackc/pgx/v5"

	"github.com/grahardi/sekolah-cbt-go/internal/httpx"
)

// MulaiUjian opens a jadwal for peserta login by creating its ujians row
// with status 'aktif'. PesertaLogin refuses to start an attempt until this
// exists, so this is the actual "start exam" switch for the whole jadwal.
// Safe to call twice — returns the existing aktif ujian instead of erroring.
func (h *Handlers) MulaiUjian(w http.ResponseWriter, r *http.Request) {
	jadwalID := r.PathValue("id")
	ctx := r.Context()

	var existing string
	err := h.Pool.QueryRow(ctx,
		`SELECT id FROM ujians WHERE jadwal_id = $1 AND status = 'aktif'`, jadwalID,
	).Scan(&existing)
	if err == nil {
		httpx.WriteJSON(w, http.StatusOK, map[string]string{"ujian_id": existing, "status": "aktif"})
		return
	}
	if err != pgx.ErrNoRows {
		httpx.WriteError(w, http.StatusInternalServerError, "gagal memeriksa ujian")
		return
	}

	var id string
	err = h.Pool.QueryRow(ctx,
		`INSERT INTO ujians (jadwal_id, status) VALUES ($1, 'aktif') RETURNING id`, jadwalID,
	).Scan(&id)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "gagal memulai ujian")
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, map[string]string{"ujian_id": id, "status": "aktif"})
}

// SelesaiUjianAdmin closes the jadwal's active ujian so no new peserta can
// log in — attempts already in progress can still finish and submit via
// POST /ujian/selesai until their token expires.
func (h *Handlers) SelesaiUjianAdmin(w http.ResponseWriter, r *http.Request) {
	jadwalID := r.PathValue("id")

	tag, err := h.Pool.Exec(r.Context(), `
		UPDATE ujians SET status = 'selesai', updated_at = now()
		WHERE jadwal_id = $1 AND status = 'aktif'
	`, jadwalID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "gagal menghentikan ujian")
		return
	}
	if tag.RowsAffected() == 0 {
		httpx.WriteError(w, http.StatusNotFound, "tidak ada ujian aktif untuk jadwal ini")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]bool{"stopped": true})
}
