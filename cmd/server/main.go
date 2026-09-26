package main

import (
	"database/sql"
	"flag"
	"log"
	"net/http"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	configPath := flag.String("config", "/etc/platform/config.json", "path to JSON config")
	flag.Parse()

	cfg, err := LoadConfig(*configPath)
	if err != nil {
		log.Fatal(err)
	}

	db, err := sql.Open("pgx", cfg.DSN())
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(4)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", HealthHandler(db))
	mux.HandleFunc("/", IndexHandler(cfg))

	srv := &http.Server{
		Addr:              cfg.Listen,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("listening on %s env=%s db_host=%s", cfg.Listen, cfg.Env, cfg.DBHost)
	log.Fatal(srv.ListenAndServe())
}
