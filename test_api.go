package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
)

func main() {
	// Test the API endpoints
	fmt.Println("Testing API on port 8081...")

	// Test profiles endpoint
	resp, err := http.Get("http://localhost:8081/api/profiles")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	fmt.Printf("GET /api/profiles - Status: %s\n", resp.Status)
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		os.Exit(1)
	}

	// Check if we got any profiles
	var profiles []interface{}
	if err := json.Unmarshal(body, &profiles); err != nil {
		fmt.Printf("Error unmarshaling profiles: %v\n", err)
		fmt.Printf("Response body: %s\n", string(body))
	} else {
		fmt.Printf("Found %d profiles\n", len(profiles))
		if len(profiles) > 0 {
			// Pretty print the first profile
			firstProfile, err := json.Marshal(profiles[0])
			if err != nil {
				fmt.Printf("Error marshaling first profile: %v\n", err)
			} else {
				var prettyJSON bytes.Buffer
				if err := json.Indent(&prettyJSON, firstProfile, "", "  "); err != nil {
					fmt.Printf("Error formatting JSON: %v\n", err)
				} else {
					fmt.Printf("First profile:\n%s\n", prettyJSON.String())
				}
			}
		}
	}

	// Test specific profile endpoint
	profileName := "Albert Einstein"
	encodedName := url.PathEscape(profileName)
	profileResp, err := http.Get("http://localhost:8081/api/profiles/" + encodedName)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer profileResp.Body.Close()

	fmt.Printf("\nGET /api/profiles/%s - Status: %s\n", encodedName, profileResp.Status)
	
	profileBody, err := io.ReadAll(profileResp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		os.Exit(1)
	}

	var profilePrettyJSON bytes.Buffer
	err = json.Indent(&profilePrettyJSON, profileBody, "", "  ")
	if err != nil {
		fmt.Printf("Error formatting JSON: %v\n", err)
		fmt.Printf("Raw response: %s\n", string(profileBody))
	} else {
		fmt.Printf("Profile response:\n%s\n", profilePrettyJSON.String())
	}

	// Test search endpoint
	searchTerm := "Einstein"
	searchResp, err := http.Get("http://localhost:8081/api/search?q=" + searchTerm)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer searchResp.Body.Close()

	fmt.Printf("\nGET /api/search?q=%s - Status: %s\n", searchTerm, searchResp.Status)
	
	searchBody, err := io.ReadAll(searchResp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		os.Exit(1)
	}

	var searchResults []interface{}
	if err := json.Unmarshal(searchBody, &searchResults); err != nil {
		fmt.Printf("Error unmarshaling search results: %v\n", err)
		fmt.Printf("Raw response: %s\n", string(searchBody))
	} else {
		fmt.Printf("Found %d search results\n", len(searchResults))
		if len(searchResults) > 0 {
			// Pretty print the search results
			resultsJSON, err := json.Marshal(searchResults)
			if err != nil {
				fmt.Printf("Error marshaling search results: %v\n", err)
			} else {
				var prettyResults bytes.Buffer
				if err := json.Indent(&prettyResults, resultsJSON, "", "  "); err != nil {
					fmt.Printf("Error formatting JSON: %v\n", err)
				} else {
					fmt.Printf("Search results:\n%s\n", prettyResults.String())
				}
			}
		}
	}
}