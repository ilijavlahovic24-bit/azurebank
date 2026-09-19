package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ilijavlahovic24-bit/azurebank/internal/accounts"
	"github.com/ilijavlahovic24-bit/azurebank/internal/auth"
)

func main() {
	ctx := context.Background()

	dbURL := os.Getenv("DATABASE_URL")
	jwtSecret := os.Getenv("JWT_SECRET")
	if dbURL == "" || jwtSecret == "" {
		log.Fatal("DATABASE_URL and JWT_SECRET must be set")
	}

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("db pool: %v", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("db ping: %v", err)
	}

	authStore := auth.NewStore(pool)
	authHandler := auth.NewHandler(authStore, jwtSecret)

	accountsStore := accounts.NewStore(pool)
	accountsHandler := accounts.NewHandler(accountsStore)

	mux := http.NewServeMux()

	// javne rute
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("POST /auth/register", authHandler.Register)
	mux.HandleFunc("POST /auth/login", authHandler.Login)

	// zaštićene rute — sub-mux sa auth middleware-om
	authMW := auth.Middleware(jwtSecret)

	protected := http.NewServeMux()
	protected.HandleFunc("GET /accounts", accountsHandler.List)
	protected.HandleFunc("POST /accounts", accountsHandler.Create)

	mux.Handle("/accounts", authMW(protected))
	mux.Handle("/accounts/", authMW(protected))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("starting azurebank on :%s", port)
	log.Fatal(srv.ListenAndServe())
}
