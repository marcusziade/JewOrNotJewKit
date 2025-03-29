package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/marcusziade/jewornotjew/pkg/client"
	"github.com/marcusziade/jewornotjew/pkg/db"
	"github.com/marcusziade/jewornotjew/pkg/models"
)

func main() {
	// Define command line flags
	dataDir := flag.String("data-dir", "./data", "Directory to store scraped data")
	dbPath := flag.String("db-path", "./jewornotjew.db", "Path to SQLite database")
	baseURL := flag.String("base-url", "http://jewornotjew.com", "Base URL to scrape")
	loadOnly := flag.Bool("load-only", false, "Only load data from disk, don't scrape")
	flag.Parse()

	// Create data directory if it doesn't exist
	if err := os.MkdirAll(*dataDir, 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	// Initialize client
	c, err := client.NewClient(
		client.WithBaseURL(*baseURL),
		client.WithDataDir(*dataDir),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Scrape or load profiles
	if *loadOnly {
		fmt.Println("Loading profiles from disk...")
		if err := c.LoadFromDisk(); err != nil {
			log.Fatalf("Failed to load profiles from disk: %v", err)
		}
	} else {
		fmt.Println("Scraping profiles...")
		if err := c.ScrapeAll(); err != nil {
			log.Fatalf("Failed to scrape profiles: %v", err)
		}
	}

	// Initialize database
	db, err := db.New(*dbPath)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize database schema
	if err := db.InitSchema(); err != nil {
		log.Fatalf("Failed to initialize database schema: %v", err)
	}

	// Check if any profiles were scraped
	profiles := c.ListProfiles()
	if len(profiles) == 0 {
		fmt.Println("No profiles scraped. Creating mock profiles for testing...")
		
		// Create mock profiles for testing
		mockProfiles := []*models.Profile{
			{
				Name:        "Albert Einstein",
				URL:         "http://jewornotjew.com/profile.jsp?ID=43",
				Verdict:     "Jew",
				Description: "Albert Einstein was a German-born theoretical physicist who developed the theory of relativity, one of the two pillars of modern physics (alongside quantum mechanics). His work is also known for its influence on the philosophy of science.",
				Pros:        []string{"Born to Jewish parents", "Identified as Jewish throughout his life", "Supported Jewish causes", "Offered presidency of Israel"},
				Cons:        []string{"Non-observant", "Rejected organized religion"},
				Category:    "Science",
				ImageURL:    "http://jewornotjew.com/images/scientists/einstein.jpg",
				CreatedAt:   time.Now().Format(time.RFC3339),
				UpdatedAt:   time.Now().Format(time.RFC3339),
			},
			{
				Name:        "Adam Sandler",
				URL:         "http://jewornotjew.com/profile.jsp?ID=1",
				Verdict:     "Jew",
				Description: "Adam Sandler is an American actor, comedian, screenwriter, film producer, and musician. Known for comedy but also received critical acclaim for dramatic roles.",
				Pros:        []string{"Born to Jewish parents", "Bar Mitzvah'd", "Created 'The Chanukah Song'", "Often references his Jewish heritage in his work"},
				Cons:        []string{"Married a non-Jew (who converted)"},
				Category:    "Entertainment",
				ImageURL:    "http://jewornotjew.com/images/actors/sandler.jpg",
				CreatedAt:   time.Now().Format(time.RFC3339),
				UpdatedAt:   time.Now().Format(time.RFC3339),
			},
			{
				Name:        "Stephen Spielberg",
				URL:         "http://jewornotjew.com/profile.jsp?ID=2",
				Verdict:     "Jew",
				Description: "Steven Spielberg is an American film director, producer, and screenwriter. He is considered one of the founding pioneers of the New Hollywood era and one of the most popular directors and producers in film history.",
				Pros:        []string{"Born to Jewish parents", "Made 'Schindler's List'", "Established the Shoah Foundation", "Openly identifies as Jewish"},
				Cons:        []string{"Once said he was ashamed of his Judaism"},
				Category:    "Entertainment",
				ImageURL:    "http://jewornotjew.com/images/directors/spielberg.jpg",
				CreatedAt:   time.Now().Format(time.RFC3339),
				UpdatedAt:   time.Now().Format(time.RFC3339),
			},
			{
				Name:        "Jesus Christ",
				URL:         "http://jewornotjew.com/profile.jsp?ID=200",
				Verdict:     "Jew",
				Description: "Jesus of Nazareth, also known as Jesus Christ, was a first-century Jewish preacher and religious leader. He is the central figure of Christianity.",
				Pros:        []string{"Born to Jewish parents", "Circumcised", "Called 'Rabbi'", "Followed Jewish law", "Last Supper was a Passover Seder"},
				Cons:        []string{"Started a religion that has historically been anti-Jewish", "Christians don't consider him Jewish"},
				Category:    "Religion",
				ImageURL:    "http://jewornotjew.com/images/religion/jesus.jpg",
				CreatedAt:   time.Now().Format(time.RFC3339),
				UpdatedAt:   time.Now().Format(time.RFC3339),
			},
			{
				Name:        "Madonna",
				URL:         "http://jewornotjew.com/profile.jsp?ID=300",
				Verdict:     "Not a Jew",
				Description: "Madonna Louise Ciccone is an American singer, songwriter, and actress. Known as the 'Queen of Pop', she is acclaimed for her continual reinvention and versatility in music production, songwriting, and visual presentation.",
				Pros:        []string{"Practices Kabbalah", "Hebrew name 'Esther'", "Visits Israel regularly"},
				Cons:        []string{"Born Catholic", "Not converted according to Jewish law", "Appropriates Jewish mysticism"},
				Category:    "Music",
				ImageURL:    "http://jewornotjew.com/images/musicians/madonna.jpg",
				CreatedAt:   time.Now().Format(time.RFC3339),
				UpdatedAt:   time.Now().Format(time.RFC3339),
			},
		}
		
		// Save mock profiles to client and JSON
		for _, profile := range mockProfiles {
			c.AddProfile(profile)
			if err := c.SaveProfileToJSON(profile); err != nil {
				log.Printf("Failed to save mock profile %s to JSON: %v", profile.Name, err)
			}
		}
		
		profiles = mockProfiles
	}
	
	// Load profiles into database
	fmt.Println("Loading profiles into database...")
	for _, profile := range profiles {
		if err := db.InsertProfile(profile); err != nil {
			log.Printf("Failed to insert profile %s: %v", profile.Name, err)
		}
	}

	fmt.Printf("Successfully processed %d profiles\n", len(profiles))
	fmt.Printf("Data stored in %s\n", *dataDir)
	fmt.Printf("Database stored at %s\n", *dbPath)

	// Print example usage
	absPath, _ := filepath.Abs(*dbPath)
	fmt.Println("\nExample usage:")
	fmt.Println("  List all profiles:")
	fmt.Printf("    go run cmd/cli/main.go -db %s list\n", absPath)
	fmt.Println("  Search for profiles:")
	fmt.Printf("    go run cmd/cli/main.go -db %s search \"Einstein\"\n", absPath)
}