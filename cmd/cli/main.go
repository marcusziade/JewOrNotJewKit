package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/marcusziade/jewornotjew/pkg/db"
	"github.com/marcusziade/jewornotjew/pkg/models"
)

// ANSI color codes
var (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorWhite  = "\033[37m"
	Bold        = "\033[1m"
)

func main() {
	// Print fancy header
	printHeader()
	
	// Define command line flags
	dbPath := flag.String("db", "./jewornotjew.db", "Path to SQLite database")
	jsonOutput := flag.Bool("json", false, "Output in JSON format")
	noColor := flag.Bool("no-color", false, "Disable colored output")
	flag.Parse()
	
	// Disable colors if requested
	if *noColor {
		disableColors()
	}

	// Check if database exists
	if _, err := os.Stat(*dbPath); os.IsNotExist(err) {
		log.Fatalf("%sDatabase file not found:%s %s\n%sRun the scraper first:%s go run cmd/scraper/main.go", 
			ColorRed+Bold, ColorReset, *dbPath, ColorYellow+Bold, ColorReset)
	}

	// Connect to database
	db, err := db.New(*dbPath)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Get command
	args := flag.Args()
	if len(args) < 1 {
		printUsage()
		os.Exit(1)
	}

	command := args[0]

	switch command {
	case "list":
		// List all profiles
		profiles, err := db.ListProfiles()
		if err != nil {
			log.Fatalf("Failed to list profiles: %v", err)
		}
		outputProfiles(profiles, *jsonOutput)

	case "search":
		// Search for profiles
		if len(args) < 2 {
			fmt.Println("Error: search command requires a query")
			printUsage()
			os.Exit(1)
		}
		query := args[1]
		profiles, err := db.SearchProfiles(query)
		if err != nil {
			log.Fatalf("Failed to search profiles: %v", err)
		}
		outputProfiles(profiles, *jsonOutput)

	case "get":
		// Get a specific profile
		if len(args) < 2 {
			fmt.Println("Error: get command requires a name")
			printUsage()
			os.Exit(1)
		}
		name := args[1]
		profile, err := db.GetProfile(name)
		if err != nil {
			log.Fatalf("Failed to get profile: %v", err)
		}
		outputProfile(profile, *jsonOutput)

	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Printf("%s%sUsage:%s\n", Bold, ColorCyan, ColorReset)
	fmt.Println("  go run cmd/cli/main.go [flags] <command> [arguments]")
	
	fmt.Printf("\n%s%sCommands:%s\n", Bold, ColorCyan, ColorReset)
	fmt.Printf("  %slist%s                  List all profiles\n", Bold, ColorReset)
	fmt.Printf("  %ssearch%s <query>        Search for profiles\n", Bold, ColorReset)
	fmt.Printf("  %sget%s <name>            Get a specific profile\n", Bold, ColorReset)
	
	fmt.Printf("\n%s%sFlags:%s\n", Bold, ColorCyan, ColorReset)
	fmt.Printf("  %s-db%s <path>            Path to SQLite database (default: ./jewornotjew.db)\n", Bold, ColorReset)
	fmt.Printf("  %s-json%s                 Output in JSON format\n", Bold, ColorReset)
	fmt.Printf("  %s-no-color%s             Disable colored output\n", Bold, ColorReset)
	
	fmt.Printf("\n%s%sExamples:%s\n", Bold, ColorCyan, ColorReset)
	fmt.Println("  go run cmd/cli/main.go list")
	fmt.Println("  go run cmd/cli/main.go search \"Einstein\"")
	fmt.Println("  go run cmd/cli/main.go get \"Leonard Nimoy\"")
}

// printHeader prints a fancy ASCII art header
func printHeader() {
	header := `
     __                ___           _   __       __                
    / /__  _    __    / _ \_______  / | / /___   / /_  ___  _    __
   / // / | |/|/ /__ / // / __/ _ \/  |/ / _ \  / __/ / _ \| |/|/ /
  / // /  |/|/|/__// __ / _// , _/ /|  / ___/ /_/   / // /|/|/|/__/
 /____/   |/|/     /_/ |_|_____/_/_/|_/_/    (_)   /____/ |/|/     
                                                                   
`
	fmt.Printf("%s%s%s\n", Bold, ColorYellow, header)
	fmt.Printf("%s%s%s%s%s%s\n", 
		ColorReset, ColorCyan, Bold, 
		"A Go SDK for JewOrNotJew.com",
		ColorReset, "\n")
}

// disableColors removes ANSI color codes
func disableColors() {
	ColorReset = ""
	ColorRed = ""
	ColorGreen = ""
	ColorYellow = ""
	ColorBlue = ""
	ColorPurple = ""
	ColorCyan = ""
	ColorWhite = ""
	Bold = ""
}

func outputProfiles(profiles []*models.Profile, jsonFormat bool) {
	if jsonFormat {
		data, err := json.MarshalIndent(profiles, "", "  ")
		if err != nil {
			log.Fatalf("Failed to marshal profiles to JSON: %v", err)
		}
		fmt.Println(string(data))
		return
	}

	if len(profiles) == 0 {
		fmt.Printf("%sNo profiles found%s\n", ColorYellow, ColorReset)
		return
	}

	fmt.Printf("%s%sFound %d profiles:%s\n\n", Bold, ColorCyan, len(profiles), ColorReset)
	
	for _, p := range profiles {
		// Determine verdict color
		verdictColor := ColorYellow
		if strings.Contains(strings.ToLower(p.Verdict), "jew") && !strings.Contains(strings.ToLower(p.Verdict), "not") {
			verdictColor = ColorGreen
		} else if strings.Contains(strings.ToLower(p.Verdict), "not") {
			verdictColor = ColorRed
		}
		
		fmt.Printf("%s%sName:%s %s\n", Bold, ColorBlue, ColorReset, p.Name)
		fmt.Printf("%s%sVerdict:%s %s%s%s\n", Bold, ColorBlue, ColorReset, verdictColor, p.Verdict, ColorReset)
		
		if p.Category != "" {
			fmt.Printf("%s%sCategory:%s %s\n", Bold, ColorBlue, ColorReset, p.Category)
		}
		
		fmt.Printf("%s---%s\n", ColorYellow, ColorReset)
	}
}

func outputProfile(profile *models.Profile, jsonFormat bool) {
	if jsonFormat {
		data, err := json.MarshalIndent(profile, "", "  ")
		if err != nil {
			log.Fatalf("Failed to marshal profile to JSON: %v", err)
		}
		fmt.Println(string(data))
		return
	}

	// Print header
	nameHeader := fmt.Sprintf("  %s  ", profile.Name)
	headerBar := strings.Repeat("=", len(nameHeader))
	fmt.Printf("\n%s%s%s\n", Bold+ColorCyan, headerBar, ColorReset)
	fmt.Printf("%s%s%s\n", Bold+ColorCyan, nameHeader, ColorReset)
	fmt.Printf("%s%s%s\n\n", Bold+ColorCyan, headerBar, ColorReset)

	// Determine verdict color
	verdictColor := ColorYellow
	if strings.Contains(strings.ToLower(profile.Verdict), "jew") && !strings.Contains(strings.ToLower(profile.Verdict), "not") {
		verdictColor = ColorGreen
	} else if strings.Contains(strings.ToLower(profile.Verdict), "not") {
		verdictColor = ColorRed
	}

	// Print basic information
	fmt.Printf("%s%sURL:%s %s\n", Bold, ColorBlue, ColorReset, profile.URL)
	fmt.Printf("%s%sVerdict:%s %s%s%s\n", Bold, ColorBlue, ColorReset, Bold+verdictColor, profile.Verdict, ColorReset)
	
	if profile.Category != "" {
		fmt.Printf("%s%sCategory:%s %s\n", Bold, ColorBlue, ColorReset, profile.Category)
	}
	
	if profile.Description != "" {
		fmt.Printf("\n%s%sDescription:%s\n%s\n", Bold, ColorPurple, ColorReset, profile.Description)
	}
	
	// Print pros with green bullets
	if len(profile.Pros) > 0 {
		fmt.Printf("\n%s%sPros:%s\n", Bold, ColorGreen, ColorReset)
		for _, pro := range profile.Pros {
			fmt.Printf("%s•%s %s\n", ColorGreen, ColorReset, pro)
		}
	}
	
	// Print cons with red bullets
	if len(profile.Cons) > 0 {
		fmt.Printf("\n%s%sCons:%s\n", Bold, ColorRed, ColorReset)
		for _, con := range profile.Cons {
			fmt.Printf("%s•%s %s\n", ColorRed, ColorReset, con)
		}
	}
	
	// Print additional info if available
	if profile.ImageURL != "" {
		fmt.Printf("\n%s%sImage URL:%s %s\n", Bold, ColorBlue, ColorReset, profile.ImageURL)
	}
	
	fmt.Println() // Extra line for spacing
}