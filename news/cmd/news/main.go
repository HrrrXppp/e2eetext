// Command news runs the news API locally (docker-compose, Vite dev) and applies migrations.
//
//	news          serve HTTP on NEWS_ADDR (default :8081)
//	news migrate  apply pending migrations and exit
package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/ekhrunov/messenger/news/internal/database"
	"github.com/ekhrunov/messenger/news/internal/news"
)

// Version is set at link time.
var Version = "dev"

func main() {
	url := os.Getenv("NEWS_DATABASE_URL")
	if url == "" {
		log.Fatal("NEWS_DATABASE_URL is required")
	}

	if len(os.Args) > 1 {
		if os.Args[1] != "migrate" {
			log.Fatalf("unknown command %q (expected: migrate)", os.Args[1])
		}
		if err := database.Migrate(url); err != nil {
			log.Fatalf("migrate: %v", err)
		}
		log.Print("migrations applied")
		return
	}

	db, err := database.Open(url, 5)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer db.Close()

	addr := os.Getenv("NEWS_ADDR")
	if addr == "" {
		addr = ":8081"
	}
	srv := &http.Server{
		Addr:              addr,
		Handler:           news.NewHandler(news.NewRepository(db), Version),
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("news listening on %s", addr)
	log.Fatal(srv.ListenAndServe())
}
