# JewOrNotJew Go SDK

A Go SDK and database for [JewOrNotJew.com](http://jewornotjew.com). This project includes:

1. A web scraper that extracts profile data from JewOrNotJew.com
2. A SQLite database to store the scraped data
3. A Go SDK for programmatic access to the data
4. A CLI tool with a colorful interface for interacting with the data
5. A REST API for serving the data over HTTP

## Features

- 🔍 **Complete Scraping**: Extracts all 3,622 profiles with names, verdicts, descriptions, pros, cons, categories, and images
- 💾 **Persistent Storage**: Saves data in both JSON files and SQLite database
- 🌈 **Beautiful CLI**: Colorful terminal interface for browsing and searching profiles
- 🚀 **REST API**: HTTP endpoints for integration with web and mobile apps
- 🛠️ **Developer SDK**: Go packages for programmatic access to the data

## Installation

### Prerequisites

- Go 1.18 or later
- SQLite3 (for database operations)

```bash
# Clone the repository
git clone https://github.com/marcusziade/jewornotjew.git
cd jewornotjew

# Install dependencies
go mod download
```

## Scraping Data

First, run the scraper to collect data from JewOrNotJew.com:

```bash
go run cmd/scraper/main.go
```

The scraper will:
1. Download all 3,622 profile data from the website with a nice progress bar
2. Save individual profiles as JSON files in the `data` directory
3. Store all profiles in a SQLite database

Options:
- `-data-dir` - Directory to store scraped data (default: `./data`)
- `-db-path` - Path to SQLite database (default: `./jewornotjew.db`)
- `-base-url` - Base URL to scrape (default: `http://jewornotjew.com`)
- `-load-only` - Only load data from disk, don't scrape

## Using the CLI

The CLI provides a colorful interface for browsing and searching profiles:

```bash
# List all profiles
go run cmd/cli/main.go list

# Search for profiles
go run cmd/cli/main.go search "Einstein"

# Get a specific profile with detailed information
go run cmd/cli/main.go get "Leonard Nimoy"

# Output in JSON format
go run cmd/cli/main.go -json search "Einstein"

# Disable colored output
go run cmd/cli/main.go -no-color list
```

## Running the API Server

The API server provides HTTP endpoints for accessing the data:

```bash
go run cmd/api/main.go
```

Options:
- `-db` - Path to SQLite database (default: `./jewornotjew.db`)
- `-addr` - HTTP server address (default: `:8081`)

### API Endpoints

- `GET /api/profiles` - List all profiles
- `GET /api/profiles/{name}` - Get a specific profile
- `GET /api/search?q={query}` - Search for profiles

## Using the SDK in Your Code

```go
package main

import (
	"fmt"
	"log"

	"github.com/marcusziade/jewornotjew/pkg/db"
)

func main() {
	// Connect to database
	db, err := db.New("./jewornotjew.db")
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Search for profiles
	profiles, err := db.SearchProfiles("Einstein")
	if err != nil {
		log.Fatalf("Failed to search profiles: %v", err)
	}

	// Print results
	for _, profile := range profiles {
		fmt.Printf("Name: %s\n", profile.Name)
		fmt.Printf("Verdict: %s\n", profile.Verdict)
		
		if len(profile.Pros) > 0 {
			fmt.Println("Pros:")
			for _, pro := range profile.Pros {
				fmt.Printf("- %s\n", pro)
			}
		}
		
		fmt.Println("---")
	}
}
```

## Database Schema

The SQLite database contains the following tables:

- `profiles` - Main profile information (name, verdict, description, etc.)
- `pros` - Pros for each profile (foreign key to profiles)
- `cons` - Cons for each profile (foreign key to profiles)

## Building Executables

```bash
# Build all executables
go build -o bin/scraper cmd/scraper/main.go
go build -o bin/cli cmd/cli/main.go
go build -o bin/api cmd/api/main.go

# Run the built executables
./bin/scraper
./bin/cli list
./bin/api
```

## Project Structure

```
jewornotjew/
├── cmd/                 # Command-line applications
│   ├── api/             # REST API server
│   ├── cli/             # Command-line interface
│   └── scraper/         # Web scraper
├── data/                # Scraped profile data (JSON files)
├── pkg/                 # Reusable packages
│   ├── api/             # API server implementation
│   ├── client/          # Web scraping client
│   ├── db/              # Database operations
│   └── models/          # Data models
├── bin/                 # Compiled binaries (not in repo)
├── go.mod               # Go module definition
└── README.md            # This file
```

## License

MIT

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Version History

- **v1.0.0** - Initial release with complete scraping of all 3,622 profiles