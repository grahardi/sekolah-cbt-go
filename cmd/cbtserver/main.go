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

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/grahardi/sekolah-cbt-go/internal/config"
	"github.com/grahardi/sekolah-cbt-go/internal/db"
	"github.com/grahardi/sekolah-cbt-go/internal/migrate"
)

func main() {
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

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("cbt-server listening on %s (sekolah_id=%s schema=%s)", addr, cfg.SekolahID, cfg.DBSchema)
	log.Fatal(http.ListenAndServe(addr, mux))
}
