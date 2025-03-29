package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/marcusziade/jewornotjew/pkg/api"
	"github.com/marcusziade/jewornotjew/pkg/db"
	"github.com/marcusziade/jewornotjew/pkg/models"
)

func main() {
	// Define command line flags
	dbPath := flag.String("db", "./jewornotjew.db", "Path to SQLite database")
	addr := flag.String("addr", ":8081", "HTTP server address")
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

	// Make sure the database is initialized
	if err := db.InitSchema(); err != nil {
		log.Printf("Warning: Failed to initialize database schema: %v", err)
	}

	// Load profiles from data directory if database is empty
	profiles, err := db.ListProfiles()
	if err != nil || len(profiles) == 0 {
		log.Println("No profiles found in database. Importing profiles from data directory...")
		if err := importProfilesFromData(db); err != nil {
			log.Printf("Warning: Failed to import profiles: %v", err)
		}
	}

	// Create and start API server
	server := api.NewServer(db)
	log.Printf("Starting API server on %s", *addr)
	log.Printf("API endpoints:\n- GET /api/profiles\n- GET /api/profiles/{name}\n- GET /api/search?q={query}")
	if err := server.ListenAndServe(*addr); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// importProfilesFromData imports profiles from the data directory into the database
func importProfilesFromData(db *db.DB) error {
	dataDir := "./data"
	files, err := os.ReadDir(dataDir)
	if err != nil {
		return fmt.Errorf("failed to read data directory: %w", err)
	}

	importCount := 0
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		filePath := filepath.Join(dataDir, file.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			log.Printf("Failed to read file %s: %v", filePath, err)
			continue
		}

		var profile models.Profile
		if err := json.Unmarshal(data, &profile); err != nil {
			log.Printf("Failed to unmarshal profile from %s: %v", filePath, err)
			continue
		}

		if err := db.InsertProfile(&profile); err != nil {
			log.Printf("Failed to insert profile %s: %v", profile.Name, err)
			continue
		}
		importCount++
	}

	log.Printf("Imported %d profiles into the database", importCount)
	return nil
}