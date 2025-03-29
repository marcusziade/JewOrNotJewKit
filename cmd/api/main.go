package main

import (
	"flag"
	"log"
	"os"

	"github.com/marcusziade/jewornotjew/pkg/api"
	"github.com/marcusziade/jewornotjew/pkg/db"
)

func main() {
	// Define command line flags
	dbPath := flag.String("db", "./jewornotjew.db", "Path to SQLite database")
	addr := flag.String("addr", ":8080", "HTTP server address")
	flag.Parse()

	// Check if database exists
	if _, err := os.Stat(*dbPath); os.IsNotExist(err) {
		log.Fatalf("Database file not found: %s\nRun the scraper first: go run cmd/scraper/main.go", *dbPath)
	}

	// Connect to database
	db, err := db.New(*dbPath)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Create and start API server
	server := api.NewServer(db)
	log.Printf("Starting API server on %s", *addr)
	log.Printf("API endpoints:\n- GET /api/profiles\n- GET /api/profiles/{name}\n- GET /api/search?q={query}")
	if err := server.ListenAndServe(*addr); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}