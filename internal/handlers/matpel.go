package handlers

import (
	"net/http"

	"github.com/grahardi/sekolah-cbt-go/internal/httpx"
)

type matpelItem struct {
	ID   string `json:"id"`
	Nama string `json:"nama"`
	Kode string `json:"kode"`
}

type matpelRequest struct {
	Nama string `json:"nama"`
	Kode string `json:"kode"`
}

func (h *Handlers) ListMatpel(w http.ResponseWriter, r *http.Request) {
	rows, err := h.Pool.Query(r.Context(), `SELECT id, nama, coalesce(kode, '') FROM matpels ORDER BY nama`)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "gagal mengambil matpel")
		return
	}
	defer rows.Close()

	var items []matpelItem
	for rows.Next() {
		var it matpelItem
		if err := rows.Scan(&it.ID, &it.Nama, &it.Kode); err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "gagal membaca matpel")
			return
		}
		items = append(items, it)
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"matpel": items})
}

func (h *Handlers) CreateMatpel(w http.ResponseWriter, r *http.Request) {
	var req matpelRequest
	if err := httpx.DecodeJSON(r, &req); err != nil || req.Nama == "" {
		httpx.WriteError(w, http.StatusBadRequest, "nama wajib diisi")
		return
	}

	var id string
	err := h.Pool.QueryRow(r.Context(),
		`INSERT INTO matpels (nama, kode) VALUES ($1, $2) RETURNING id`,
		req.Nama, httpx.NullIfEmpty(req.Kode),
	).Scan(&id)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "gagal membuat matpel")
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, matpelItem{ID: id, Nama: req.Nama, Kode: req.Kode})
}
