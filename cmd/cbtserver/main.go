// Command cbtserver is one CBT (ujian) instance for one sekolah. It's meant
// to be provisioned exactly like the current Extraordinary CBT instances:
// one copy of this same binary per sekolah, its own .env (DB_SCHEMA, PORT,
// SEKOLAH_ID), its own systemd unit, reverse-proxied by its own Nginx vhost.
//
//	cbt-server migrate   applies pending migrations for this instance's schema, then exits
//	cbt-server            starts the HTTP server
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/grahardi/sekolah-cbt-go/internal/auth"
	"github.com/grahardi/sekolah-cbt-go/internal/config"
	"github.com/grahardi/sekolah-cbt-go/internal/db"
	"github.com/grahardi/sekolah-cbt-go/internal/handlers"
	"github.com/grahardi/sekolah-cbt-go/internal/migrate"
)

func main() {
	// hash-password doesn't need the DB or a full config — handle it first
	// so it's usable for seeding test pesertas by hand via psql.
	if len(os.Args) > 2 && os.Args[1] == "hash-password" {
		hash, err := bcrypt.GenerateFromPassword([]byte(os.Args[2]), bcrypt.DefaultCost)
		if err != nil {
			log.Fatalf("hash password: %v", err)
		}
		fmt.Println(string(hash))
		return
	}

	// dev-admin-token prints a throwaway admin JWT for testing the /admin/*
	// API by hand before Laravel actually issues these. NOT for production
	// use — Laravel is the only real issuer once SSO is wired up.
	if len(os.Args) > 1 && os.Args[1] == "dev-admin-token" {
		if err := config.LoadDotEnv(".env"); err != nil {
			log.Fatalf("load .env: %v", err)
		}
		cfg := config.Load()
		claims := auth.AdminClaims{
			Typ:       "admin",
			Role:      "admin",
			SekolahID: cfg.SekolahID,
			RegisteredClaims: jwt.RegisteredClaims{
				Subject:   "dev-admin",
				IssuedAt:  jwt.NewNumericDate(time.Now()),
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			},
		}
		token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(cfg.JWTSecret))
		if err != nil {
			log.Fatalf("sign token: %v", err)
		}
		fmt.Println(token)
		return
	}

	if err := config.LoadDotEnv(".env"); err != nil {
		log.Fatalf("load .env: %v", err)
	}
	cfg := config.Load()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := db.Connect(ctx, cfg)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		if err := migrate.Run(ctx, pool, cfg.DBSchema); err != nil {
			log.Fatalf("migrate: %v", err)
		}
		fmt.Println("migrations up to date")
		return
	}

	serve(cfg, pool)
}

func serve(cfg config.Config, pool *pgxpool.Pool) {
	h := &handlers.Handlers{Pool: pool, JWTSecret: cfg.JWTSecret, SekolahID: cfg.SekolahID}

	mux := http.NewServeMux()

	// Health check for the provisioning script / uptime monitoring, same
	// role as the Run/Stop lsof port-check on the Server Ujian page today.
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		status := "ok"
		statusCode := http.StatusOK
		if err := pool.Ping(ctx); err != nil {
			status = "db_unavailable"
			statusCode = http.StatusServiceUnavailable
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(map[string]string{
			"status":     status,
			"sekolah_id": cfg.SekolahID,
			"schema":     cfg.DBSchema,
		})
	})

	mux.HandleFunc("POST /peserta/login", h.PesertaLogin)
	mux.HandleFunc("GET /ujian/soal", h.RequirePeserta(h.ListSoal))
	mux.HandleFunc("POST /ujian/jawab", h.RequirePeserta(h.SubmitJawaban))
	mux.HandleFunc("POST /ujian/selesai", h.RequirePeserta(h.SelesaiUjian))

	// Admin routes: JWT issued by Laravel, only verified here (typ=admin,
	// sekolah_id must match this instance's SEKOLAH_ID).
	mux.HandleFunc("GET /admin/matpels", h.RequireAdmin(h.ListMatpel))
	mux.HandleFunc("POST /admin/matpels", h.RequireAdmin(h.CreateMatpel))

	mux.HandleFunc("GET /admin/banksoals", h.RequireAdmin(h.ListBanksoal))
	mux.HandleFunc("POST /admin/banksoals", h.RequireAdmin(h.CreateBanksoal))
	mux.HandleFunc("PUT /admin/banksoals/{id}", h.RequireAdmin(h.UpdateBanksoal))
	mux.HandleFunc("DELETE /admin/banksoals/{id}", h.RequireAdmin(h.DeleteBanksoal))

	mux.HandleFunc("GET /admin/banksoals/{id}/soals", h.RequireAdmin(h.ListSoalAdmin))
	mux.HandleFunc("POST /admin/banksoals/{id}/soals", h.RequireAdmin(h.CreateSoal))
	mux.HandleFunc("PUT /admin/soals/{id}", h.RequireAdmin(h.UpdateSoal))
	mux.HandleFunc("DELETE /admin/soals/{id}", h.RequireAdmin(h.DeleteSoal))

	mux.HandleFunc("POST /admin/soals/{id}/jawaban", h.RequireAdmin(h.CreateJawabanSoal))
	mux.HandleFunc("PUT /admin/jawaban/{id}", h.RequireAdmin(h.UpdateJawabanSoal))
	mux.HandleFunc("DELETE /admin/jawaban/{id}", h.RequireAdmin(h.DeleteJawabanSoal))

	mux.HandleFunc("GET /admin/banksoals/{id}/jadwals", h.RequireAdmin(h.ListJadwal))
	mux.HandleFunc("POST /admin/banksoals/{id}/jadwals", h.RequireAdmin(h.CreateJadwal))
	mux.HandleFunc("PUT /admin/jadwals/{id}", h.RequireAdmin(h.UpdateJadwal))
	mux.HandleFunc("DELETE /admin/jadwals/{id}", h.RequireAdmin(h.DeleteJadwal))

	mux.HandleFunc("POST /admin/jadwals/{id}/tokens", h.RequireAdmin(h.CreateToken))
	mux.HandleFunc("PATCH /admin/tokens/{id}", h.RequireAdmin(h.SetTokenStatus))

	mux.HandleFunc("POST /admin/jadwals/{id}/mulai", h.RequireAdmin(h.MulaiUjian))
	mux.HandleFunc("POST /admin/jadwals/{id}/selesai", h.RequireAdmin(h.SelesaiUjianAdmin))

	mux.HandleFunc("GET /admin/jadwals/{id}/hasil", h.RequireAdmin(h.ListHasil))

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("cbt-server listening on %s (sekolah_id=%s schema=%s)", addr, cfg.SekolahID, cfg.DBSchema)
	log.Fatal(http.ListenAndServe(addr, mux))
}
