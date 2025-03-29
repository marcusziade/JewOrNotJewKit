package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/marcusziade/jewornotjew/pkg/models"
)

func main() {
	// Read a profile directly from the data directory
	dataDir := "./data"
	files, err := os.ReadDir(dataDir)
	if err != nil {
		fmt.Printf("Error reading data directory: %v\n", err)
		os.Exit(1)
	}

	if len(files) == 0 {
		fmt.Println("No profiles found in data directory")
		os.Exit(1)
	}

	// Find a specific profile or use the first one
	var fileName string
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".json" {
			if filepath.Base(file.Name()) == "Albert%20Einstein.json" {
				fileName = file.Name()
				break
			}
		}
	}

	if fileName == "" && len(files) > 0 {
		for _, file := range files {
			if !file.IsDir() && filepath.Ext(file.Name()) == ".json" {
				fileName = file.Name()
				break
			}
		}
	}

	if fileName == "" {
		fmt.Println("No JSON files found in data directory")
		os.Exit(1)
	}

	// Read and display the profile
	filePath := filepath.Join(dataDir, fileName)
	fmt.Printf("Reading profile from: %s\n", filePath)

	data, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}

	var profile models.Profile
	if err := json.Unmarshal(data, &profile); err != nil {
		fmt.Printf("Error unmarshaling profile: %v\n", err)
		os.Exit(1)
	}

	// Display profile details
	fmt.Println("Profile details:")
	fmt.Printf("Name: %s\n", profile.Name)
	fmt.Printf("URL: %s\n", profile.URL)
	fmt.Printf("Verdict: %s\n", profile.Verdict)
	fmt.Printf("Category: %s\n", profile.Category)
	
	if len(profile.Description) > 200 {
		fmt.Printf("Description: %s...\n", profile.Description[:200])
	} else {
		fmt.Printf("Description: %s\n", profile.Description)
	}
	
	fmt.Printf("Pros (%d): %v\n", len(profile.Pros), profile.Pros[:min(3, len(profile.Pros))])
	fmt.Printf("Cons (%d): %v\n", len(profile.Cons), profile.Cons[:min(3, len(profile.Cons))])
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}