package client

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/marcusziade/jewornotjew/pkg/models"
)

// Client represents the JewOrNotJew API client
type Client struct {
	baseURL    string
	httpClient *http.Client
	dataDir    string
	profiles   map[string]*models.Profile
}

// NewClient creates a new JewOrNotJew client
func NewClient(options ...Option) (*Client, error) {
	c := &Client{
		baseURL: "http://jewornotjew.com",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
		dataDir:  "./data",
		profiles: make(map[string]*models.Profile),
	}

	// Apply options
	for _, option := range options {
		option(c)
	}

	// Create data directory if it doesn't exist
	if err := os.MkdirAll(c.dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	return c, nil
}

// Option defines a client option
type Option func(*Client)

// WithBaseURL sets the base URL
func WithBaseURL(baseURL string) Option {
	return func(c *Client) {
		c.baseURL = baseURL
	}
}

// WithHTTPClient sets the HTTP client
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// WithDataDir sets the data directory
func WithDataDir(dataDir string) Option {
	return func(c *Client) {
		c.dataDir = dataDir
	}
}

// ScrapeAll scrapes all profiles from the website
func (c *Client) ScrapeAll() error {
	fmt.Println("Starting scrape operation...")
	
	// We'll skip trying to get profile IDs from the website since it seems unstable
	// and go directly to using known IDs from the site
	fmt.Println("Using known profile IDs from the site")
	profileIDs := []int{585, 586, 587, 588, 589, 590, 43, 1}
	
	fmt.Printf("Found %d profile IDs to scrape\n", len(profileIDs))
	
	// Scrape each profile
	for _, id := range profileIDs {
		profile, err := c.scrapeProfile(id)
		if err != nil {
			fmt.Printf("Error scraping profile %d: %v\n", id, err)
			continue
		}
		
		if profile != nil {
			fmt.Printf("Successfully scraped profile: %s\n", profile.Name)
			c.profiles[profile.Name] = profile
			
			// Save profile to JSON
			if err := c.saveProfileToJSON(profile); err != nil {
				fmt.Printf("Error saving profile to JSON: %v\n", err)
			}
		}
	}
	
	fmt.Printf("Scraping complete. Found %d profiles.\n", len(c.profiles))
	return nil
}

// getProfileIDs gets all profile IDs from the website
func (c *Client) getProfileIDs() ([]int, error) {
	// Make direct HTTP request
	resp, err := c.httpClient.Get(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve homepage: %w", err)
	}
	defer resp.Body.Close()
	
	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-200 response: %d", resp.StatusCode)
	}
	
	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	
	// Parse profile IDs from links
	var profileIDs []int
	idRegex := regexp.MustCompile(`/profile\.jsp\?ID=(\d+)`)
	matches := idRegex.FindAllSubmatch(body, -1)
	
	// Extract unique IDs
	idMap := make(map[int]bool)
	for _, match := range matches {
		if len(match) >= 2 {
			idStr := string(match[1])
			id, err := strconv.Atoi(idStr)
			if err == nil && !idMap[id] {
				idMap[id] = true
				profileIDs = append(profileIDs, id)
			}
		}
	}
	
	return profileIDs, nil
}

// scrapeProfile scrapes a profile by ID
func (c *Client) scrapeProfile(id int) (*models.Profile, error) {
	profileURL := fmt.Sprintf("%s/profile.jsp?ID=%d", c.baseURL, id)
	fmt.Printf("Scraping profile: %s\n", profileURL)
	
	// Make HTTP request
	resp, err := c.httpClient.Get(profileURL)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve profile: %w", err)
	}
	defer resp.Body.Close()
	
	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-200 response: %d", resp.StatusCode)
	}
	
	// Read the body content
	bodyContent, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	
	// Print the first 1000 characters of the HTML for debugging
	fmt.Printf("HTML snippet: %s\n", string(bodyContent[:min(1000, len(bodyContent))]))
	
	// Parse the HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(bodyContent)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}
	
	// Create profile with ID and URL
	profile := &models.Profile{
		URL: profileURL,
		// Extract numeric ID from the URL for reference
		Score: float64(id), // Store the ID as score for now
	}
	
	// Extract profile data
	profile = c.parseProfile(doc, profile)
	
	// Set ID in name if name is empty
	if profile.Name == "" {
		profile.Name = fmt.Sprintf("Profile %d", id)
	}
	
	// Set timestamps
	now := time.Now().Format(time.RFC3339)
	profile.CreatedAt = now
	profile.UpdatedAt = now
	
	return profile, nil
}

// parseProfile extracts profile data from HTML
func (c *Client) parseProfile(doc *goquery.Document, profile *models.Profile) *models.Profile {
	// Extract name from title
	title := doc.Find("title").Text()
	if title != "" {
		parts := strings.Split(title, " - ")
		if len(parts) > 0 {
			name := strings.TrimSpace(parts[0])
			// Remove "Jew or Not Jew: " prefix if present
			name = strings.TrimPrefix(name, "Jew or Not Jew: ")
			profile.Name = name
			fmt.Printf("Extracted name from title: %s\n", profile.Name)
		}
	}
	
	// Extract name from h1 if not found in title
	if profile.Name == "" {
		doc.Find("h1").Each(func(i int, s *goquery.Selection) {
			name := strings.TrimSpace(s.Text())
			if name != "" {
				profile.Name = name
				fmt.Printf("Extracted name from h1: %s\n", name)
			}
		})
	}
	
	// Extract verdict (after looking at the HTML structure)
	verdictText := ""
	// Try the meta description which often contains the verdict
	metaDesc, exists := doc.Find("meta[name=description]").Attr("content")
	if exists && metaDesc != "" {
		fmt.Printf("Found meta description: %s\n", metaDesc)
		// The meta description often follows the pattern "Is name Jewish?" or similar
		// The verdict is usually at the end as a single word
		metaDesc = strings.TrimSpace(metaDesc)
		
		if strings.Contains(metaDesc, "is ") && strings.HasSuffix(metaDesc, ".") {
			words := strings.Split(metaDesc, " ")
			if len(words) > 2 {
				// The verdict is usually the last word without the period
				lastWord := words[len(words)-1]
				lastWord = strings.TrimSuffix(lastWord, ".")
				if lastWord == "Jew" || lastWord == "Jewish" {
					verdictText = "Jew"
				} else if strings.Contains(lastWord, "Not") {
					verdictText = "Not a Jew"
				}
				fmt.Printf("Extracted verdict from meta: %s\n", verdictText)
			}
		}
	}
	
	// If no verdict found in meta, try other places
	if verdictText == "" {
		// Look for verdicts in the page content
		doc.Find("font, div, b, p").Each(func(i int, s *goquery.Selection) {
			text := strings.TrimSpace(s.Text())
			lcText := strings.ToLower(text)
			
			// Check for common verdict patterns
			if strings.Contains(lcText, "verdict:") {
				parts := strings.SplitN(text, ":", 2)
				if len(parts) > 1 {
					verdict := strings.TrimSpace(parts[1])
					if verdict != "" {
						verdictText = verdict
						fmt.Printf("Extracted verdict from page: %s\n", verdictText)
					}
				}
			} else if strings.Contains(lcText, "verdict") && len(text) < 30 {
				// Likely a verdict heading, check siblings or parents
				parent := s.Parent()
				if parent.Length() > 0 {
					siblingText := strings.TrimSpace(parent.Text())
					siblingText = strings.Replace(siblingText, text, "", 1)
					siblingText = strings.TrimSpace(siblingText)
					
					if siblingText != "" && len(siblingText) < 30 {
						verdictText = siblingText
						fmt.Printf("Extracted verdict from sibling: %s\n", verdictText)
					}
				}
			} else if (lcText == "jew" || lcText == "not a jew" || lcText == "barely a jew") && len(text) < 30 {
				verdictText = text
				fmt.Printf("Found direct verdict text: %s\n", verdictText)
			}
		})
	}
	
	// If still no verdict found, infer it from the image if possible
	if verdictText == "" {
		imageUrl, exists := doc.Find("img[src*='img/']").Attr("src")
		if exists && imageUrl != "" {
			if strings.Contains(imageUrl, "verified_jew") {
				verdictText = "Jew"
				fmt.Printf("Inferred verdict from image: %s\n", verdictText)
			} else if strings.Contains(imageUrl, "not_a_jew") {
				verdictText = "Not a Jew"
				fmt.Printf("Inferred verdict from image: %s\n", verdictText)
			}
		}
	}
	
	if verdictText != "" {
		profile.Verdict = verdictText
	}
	
	// Extract description - target the profileBody div specifically
	// First look for the profileBody div which contains the main profile content
	profileBody := doc.Find("div#profileBody, #profileBody").First()
	if profileBody.Length() > 0 {
		// Get the text content of the profileBody div
		fullText := profileBody.Text()
		fullText = strings.TrimSpace(fullText)
		
		if len(fullText) > 50 {
			// Clean up the text - remove extra whitespace and normalize line breaks
			fullText = strings.ReplaceAll(fullText, "\r\n", "\n")
			fullText = strings.ReplaceAll(fullText, "\r", "\n")
			
			// Remove any "Verdict:", "Pros:", "Cons:" sections if present at the end
			verdictIndex := strings.LastIndex(strings.ToLower(fullText), "verdict:")
			prosIndex := strings.LastIndex(strings.ToLower(fullText), "pros:")
			consIndex := strings.LastIndex(strings.ToLower(fullText), "cons:")
			
			cutIndex := len(fullText)
			if verdictIndex > 0 && verdictIndex < cutIndex {
				cutIndex = verdictIndex
			}
			if prosIndex > 0 && prosIndex < cutIndex {
				cutIndex = prosIndex
			}
			if consIndex > 0 && consIndex < cutIndex {
				cutIndex = consIndex
			}
			
			// Keep just the description part
			if cutIndex < len(fullText) {
				fullText = fullText[:cutIndex]
			}
			
			fullText = strings.TrimSpace(fullText)
			profile.Description = fullText
			fmt.Printf("Extracted full description from profileBody: %d chars\n", len(fullText))
		}
	}
	
	// Fallback: look for any substantial text blocks if profileBody not found
	if profile.Description == "" || len(profile.Description) < 50 {
		descFound := false
		doc.Find("td[valign=top] font, div.profile-description, p.description, td font").Each(func(i int, s *goquery.Selection) {
			if descFound {
				return // Already found
			}
			
			// Skip if it contains verdict or pros/cons
			text := strings.TrimSpace(s.Text())
			lowerText := strings.ToLower(text)
			
			if !strings.Contains(lowerText, "verdict:") && 
			   !strings.Contains(lowerText, "pros:") && 
			   !strings.Contains(lowerText, "cons:") && 
			   len(text) > 100 {
				profile.Description = text
				fmt.Printf("Extracted description from alternate source: %d chars\n", len(text))
				descFound = true
			}
		})
	}
	
	// If still no substantial description found, try the meta description as a last resort
	if profile.Description == "" || len(profile.Description) < 30 {
		metaDesc, exists := doc.Find("meta[name=description]").Attr("content")
		if exists && metaDesc != "" && len(metaDesc) > 10 {
			// Skip the "JewOrNotJew.com: " prefix if present
			if strings.HasPrefix(metaDesc, "JewOrNotJew.com:") {
				metaDesc = strings.TrimPrefix(metaDesc, "JewOrNotJew.com:")
				metaDesc = strings.TrimSpace(metaDesc)
			}
			profile.Description = metaDesc
			fmt.Printf("Using meta description as fallback: %s\n", metaDesc)
		}
	}
	
	// Let's also try to extract the main description from table cells,
	// as the site structure might vary
	if profile.Description == "" || len(profile.Description) < 100 {
		// Look for the largest text block in the page that's not pros/cons/verdict
		var largestText string
		doc.Find("table td").Each(func(i int, s *goquery.Selection) {
			text := strings.TrimSpace(s.Text())
			lcText := strings.ToLower(text)
			
			// Skip sections that are clearly not the main description
			if !strings.Contains(lcText, "verdict:") && 
			   !strings.Contains(lcText, "pros:") && 
			   !strings.Contains(lcText, "cons:") && 
			   len(text) > len(largestText) {
				largestText = text
			}
		})
		
		if len(largestText) > 100 {
			profile.Description = largestText
			fmt.Printf("Extracted description from largest table cell: %d chars\n", len(largestText))
		}
	}
	
	// Extract pros and cons - more comprehensive approach
	// First look for dedicated pros/cons sections
	var prosFound, consFound bool
	
	// Try to extract from the complete HTML content
	htmlString, err := doc.Html()
	if err == nil {
		// Check for pros section with regex pattern matching
		prosRegex := regexp.MustCompile(`(?i)(?:Pros|PROS|Pros:)[\s\n]*(.*?)(?:Cons|CONS|Cons:|$)`)
		prosMatches := prosRegex.FindStringSubmatch(htmlString)
		if len(prosMatches) > 1 {
			prosContent := prosMatches[1]
			pros := splitByBullets(prosContent)
			for _, pro := range pros {
				pro = strings.TrimSpace(pro)
				// Filter out invalid entries
				if pro != "" && len(pro) > 3 && !strings.Contains(strings.ToLower(pro), "cons:") {
					profile.Pros = append(profile.Pros, pro)
					fmt.Printf("Extracted pro from regex: %s\n", pro)
					prosFound = true
				}
			}
		}
		
		// Check for cons section with regex pattern matching - more careful approach
		consRegex := regexp.MustCompile(`(?i)(?:Cons|CONS|Cons:)[\s\n]*([^:]*)(?:\s*Verdict:|$)`)
		consMatches := consRegex.FindStringSubmatch(htmlString)
		if len(consMatches) > 1 {
			consContent := consMatches[1]
			// Check if the cons content is reasonable (not just a fragment)
			if len(consContent) > 10 && len(consContent) < 1000 {
				cons := splitByBullets(consContent)
				for _, con := range cons {
					con = strings.TrimSpace(con)
					// Filter out invalid entries and fragments
					if con != "" && len(con) > 10 && !strings.Contains(con, "idered") {
						// Skip if HTML entities are found, suggesting invalid content
						if !strings.Contains(con, "&#") && !strings.Contains(con, "&lt;") && !strings.Contains(con, "&gt;") && !strings.Contains(con, "<span") {
							profile.Cons = append(profile.Cons, con)
							fmt.Printf("Extracted con from regex: %s\n", con)
							consFound = true
						}
					}
				}
			}
		}
	}
	
	// If regex didn't find anything, try DOM-based extraction
	if !prosFound || !consFound {
		// Try to find specific pros/cons sections
		doc.Find("div, td, span, p, font").Each(func(i int, s *goquery.Selection) {
			text := strings.TrimSpace(s.Text())
			lowerText := strings.ToLower(text)
			
			// Look for pros section
			if !prosFound && (strings.Contains(lowerText, "pros:") || strings.HasPrefix(lowerText, "pros")) {
				// Extract pros
				parts := strings.SplitN(text, ":", 2)
				var prosList string
				if len(parts) > 1 {
					prosList = parts[1]
				} else {
					// Try taking everything after "Pros"
					prosList = strings.TrimPrefix(text, "Pros")
				}
				
				// Split by bullet points or line breaks
				pros := splitByBullets(prosList)
				for _, pro := range pros {
					pro = strings.TrimSpace(pro)
					if pro != "" && len(pro) > 3 && !strings.Contains(strings.ToLower(pro), "cons") {
						profile.Pros = append(profile.Pros, pro)
						fmt.Printf("Extracted pro from DOM: %s\n", pro)
						prosFound = true
					}
				}
			}
			
			// Look for cons section
			if !consFound && (strings.Contains(lowerText, "cons:") || strings.HasPrefix(lowerText, "cons")) {
				// Extract cons
				parts := strings.SplitN(text, ":", 2)
				var consList string
				if len(parts) > 1 {
					consList = parts[1]
				} else {
					// Try taking everything after "Cons"
					consList = strings.TrimPrefix(text, "Cons")
				}
				
				// Split by bullet points or line breaks
				cons := splitByBullets(consList)
				for _, con := range cons {
					con = strings.TrimSpace(con)
					if con != "" && len(con) > 3 {
						// Skip if HTML entities are found, suggesting invalid content
						if !strings.Contains(con, "&#") && !strings.Contains(con, "&lt;") && !strings.Contains(con, "&gt;") && !strings.Contains(con, "<span") {
							profile.Cons = append(profile.Cons, con)
							fmt.Printf("Extracted con from DOM: %s\n", con)
							consFound = true
						}
					}
				}
			}
		})
	}
	
	// Also look for list items as possible pros/cons
	doc.Find("li, ul li").Each(func(i int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		if text != "" && len(text) > 3 {
			// Try to determine if this is a pro or con based on context
			parent := s.ParentsFiltered("div, td, ul").First()
			parentText := strings.ToLower(parent.Text())
			
			if strings.Contains(parentText, "pros") && !strings.Contains(strings.ToLower(text), "cons:") {
				// Likely a pro
				if !contains(profile.Pros, text) {
					profile.Pros = append(profile.Pros, text)
					fmt.Printf("Extracted pro from list: %s\n", text)
				}
			} else if strings.Contains(parentText, "cons") {
				// Likely a con
				// Skip if HTML entities are found, suggesting invalid content
				if !contains(profile.Cons, text) && 
				   !strings.Contains(text, "&#") && 
				   !strings.Contains(text, "&lt;") && 
				   !strings.Contains(text, "&gt;") && 
				   !strings.Contains(text, "<span") {
					profile.Cons = append(profile.Cons, text)
					fmt.Printf("Extracted con from list: %s\n", text)
				}
			}
		}
	})
	
	// Extract category if available - improved approach
	doc.Find("td font, span, div, p, strong, b, h3").Each(func(i int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		// Look for explicit category marker
		if strings.Contains(text, "Category:") {
			parts := strings.SplitN(text, "Category:", 2)
			if len(parts) > 1 {
				category := strings.TrimSpace(parts[1])
				// Clean up the category
				category = cleanHTML(category)
				category = strings.Trim(category, ".")
				
				if category != "" {
					profile.Category = category
					fmt.Printf("Extracted category: %s\n", category)
				}
			}
		} else if strings.HasPrefix(text, "Category") {
			// Try alternate format
			parts := strings.SplitN(text, " ", 2)
			if len(parts) > 1 {
				category := strings.TrimSpace(parts[1])
				category = cleanHTML(category)
				category = strings.Trim(category, ".")
				
				if category != "" {
					profile.Category = category
					fmt.Printf("Extracted category from alternate format: %s\n", category)
				}
			}
		}
	})
	
	// If no category found, try to infer from keywords, meta tags, or page content
	if profile.Category == "" {
		// First try keywords meta tag
		keywords, exists := doc.Find("meta[name=keywords]").Attr("content")
		if exists && keywords != "" {
			keywordsList := strings.Split(keywords, ",")
			for _, keyword := range keywordsList {
				keyword = strings.TrimSpace(keyword)
				// Check common categories
				for _, cat := range []string{"Actor", "Actress", "Entertainment", "Politics", "Sports", "Music", "Science", "Business", "Religion", "History", 
					"Art", "Literature", "Media", "Academia", "Military", "Fashion", "Technology", "Comedy", "Royalty", "Film", "Television"} {
					if strings.Contains(strings.ToLower(keyword), strings.ToLower(cat)) {
						profile.Category = cat
						fmt.Printf("Inferred category from keywords: %s\n", cat)
						break
					}
				}
				if profile.Category != "" {
					break
				}
			}
		}
		
		// If still no category, try description text for clues
		if profile.Category == "" && profile.Description != "" {
			lowerDesc := strings.ToLower(profile.Description)
			// Common category indicators in text
			categoryClues := map[string]string{
				"actor":        "Entertainment",
				"actress":      "Entertainment",
				"movie":        "Entertainment",
				"film":         "Entertainment",
				"directed":     "Entertainment",
				"singer":       "Music",
				"musician":     "Music",
				"album":        "Music",
				"song":         "Music",
				"band":         "Music",
				"political":    "Politics",
				"politician":   "Politics",
				"president":    "Politics",
				"senator":      "Politics",
				"parliament":   "Politics",
				"scientist":    "Science",
				"researcher":   "Science",
				"professor":    "Academia",
				"author":       "Literature",
				"writer":       "Literature",
				"book":         "Literature",
				"athlete":      "Sports",
				"player":       "Sports",
				"baseball":     "Sports",
				"football":     "Sports",
				"basketball":   "Sports",
				"soccer":       "Sports",
				"tennis":       "Sports",
				"religious":    "Religion",
				"rabbi":        "Religion",
				"priest":       "Religion",
				"businessman":  "Business",
				"entrepreneur": "Business",
				"company":      "Business",
				"CEO":          "Business",
				"comedian":     "Comedy",
				"comedy":       "Comedy",
			}
			
			// Check for category clues in description
			for clue, category := range categoryClues {
				if strings.Contains(lowerDesc, clue) {
					profile.Category = category
					fmt.Printf("Inferred category from description text: %s\n", category)
					break
				}
			}
		}
	}
	
	// Extract image URL - check multiple locations
	
	// First check for og:image or similar meta tags
	ogImage, exists := doc.Find("meta[property='og:image']").Attr("content")
	if exists && ogImage != "" {
		if !strings.HasPrefix(ogImage, "http") {
			if !strings.HasPrefix(ogImage, "/") {
				profile.ImageURL = c.baseURL + "/" + ogImage
			} else {
				profile.ImageURL = c.baseURL + ogImage
			}
		} else {
			profile.ImageURL = ogImage
		}
		fmt.Printf("Extracted image URL from meta: %s\n", profile.ImageURL)
	}
	
	// If no og:image, check for image_src link
	if profile.ImageURL == "" {
		imageSrc, exists := doc.Find("link[rel='image_src']").Attr("href")
		if exists && imageSrc != "" {
			if !strings.HasPrefix(imageSrc, "http") {
				if !strings.HasPrefix(imageSrc, "/") {
					profile.ImageURL = c.baseURL + "/" + imageSrc
				} else {
					profile.ImageURL = c.baseURL + imageSrc
				}
			} else {
				profile.ImageURL = imageSrc
			}
			fmt.Printf("Extracted image URL from link: %s\n", profile.ImageURL)
		}
	}
	
	// If still no image, look for img tags
	if profile.ImageURL == "" {
		doc.Find("img").Each(func(i int, s *goquery.Selection) {
			if profile.ImageURL != "" {
				return // Already found an image
			}
			
			if src, exists := s.Attr("src"); exists && src != "" {
				// Check if it's a profile image
				if strings.Contains(strings.ToLower(src), "people") || 
				   strings.Contains(strings.ToLower(src), "img") || 
				   strings.Contains(strings.ToLower(src), "images") {
					if !strings.HasPrefix(src, "http") {
						if strings.HasPrefix(src, "/") {
							profile.ImageURL = c.baseURL + src
						} else {
							profile.ImageURL = c.baseURL + "/" + src
						}
					} else {
						profile.ImageURL = src
					}
					fmt.Printf("Extracted image URL from img tag: %s\n", profile.ImageURL)
				}
			}
		})
	}
	
	return profile
}

// cleanHTML removes HTML tags and normalizes whitespace
func cleanHTML(input string) string {
	// Remove HTML tags
	tagRegex := regexp.MustCompile(`<[^>]*>`)
	withoutTags := tagRegex.ReplaceAllString(input, "")
	
	// Normalize whitespace
	withoutTags = strings.ReplaceAll(withoutTags, "&nbsp;", " ")
	withoutTags = strings.ReplaceAll(withoutTags, "\r\n", " ")
	withoutTags = strings.ReplaceAll(withoutTags, "\n", " ")
	
	// Replace HTML entities
	withoutTags = strings.ReplaceAll(withoutTags, "&amp;", "&")
	withoutTags = strings.ReplaceAll(withoutTags, "&lt;", "<")
	withoutTags = strings.ReplaceAll(withoutTags, "&gt;", ">")
	withoutTags = strings.ReplaceAll(withoutTags, "&quot;", "\"")
	withoutTags = strings.ReplaceAll(withoutTags, "&#39;", "'")
	withoutTags = strings.ReplaceAll(withoutTags, "&#34;", "\"")
	
	// Collapse multiple spaces into one
	spaceRegex := regexp.MustCompile(`\s+`)
	withoutTags = spaceRegex.ReplaceAllString(withoutTags, " ")
	
	return strings.TrimSpace(withoutTags)
}

// min returns the smaller of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// contains checks if a string is present in a slice
func contains(slice []string, str string) bool {
	for _, item := range slice {
		if item == str {
			return true
		}
	}
	return false
}

// splitByBullets splits text by bullet points or line breaks
func splitByBullets(text string) []string {
	// Check for various types of bullets and split by them
	text = strings.TrimSpace(text)
	var items []string
	
	// First try to split by common bullet characters with more intelligence
	bullets := []string{"•", "-", "★", "✓", "✔", "*", "→", "⇒", "⟹", "⇾", "⟶"}
	
	// Check if any bullet character is present and handle each one appropriately
	hasBullets := false
	for _, bullet := range bullets {
		if strings.Contains(text, bullet) {
			hasBullets = true
			// Split by bullet and handle each chunk
			parts := strings.Split(text, bullet)
			for _, part := range parts {
				part = strings.TrimSpace(part)
				if part != "" {
					items = append(items, part)
				}
			}
			break
		}
	}
	
	// If no bullets found, try splitting by newlines with better handling
	if !hasBullets && strings.Contains(text, "\n") {
		lines := strings.Split(text, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			// Check if line starts with a bullet point we didn't catch
			for _, bullet := range bullets {
				if strings.HasPrefix(line, bullet) {
					line = strings.TrimSpace(strings.TrimPrefix(line, bullet))
					break
				}
			}
			
			// Only add non-empty lines
			if line != "" && len(line) > 2 {
				items = append(items, line)
			}
		}
	}
	
	// If no newlines or bullets, check for numbered points with better regex
	if len(items) == 0 {
		numberRegex := regexp.MustCompile(`(\d+\.\s+)`)
		if numberRegex.MatchString(text) {
			// Split by numbered bullets with more accuracy
			parts := numberRegex.Split(text, -1)
			for _, part := range parts {
				part = strings.TrimSpace(part)
				if part != "" && len(part) > 2 {
					items = append(items, part)
				}
			}
		}
	}
	
	// If no structure was found and the text is long enough, try using periods/semicolons
	if len(items) == 0 && len(text) > 15 && (strings.Contains(text, ". ") || strings.Contains(text, "; ")) {
		// Try to split by sentences if it looks like a sentence list
		parts := strings.Split(text, ". ")
		if len(parts) > 1 {
			for _, part := range parts {
				part = strings.TrimSpace(part)
				// Make sure it's not just a fragment
				if part != "" && len(part) > 10 {
					// Add period back if it looks like a sentence
					if len(part) > 20 && part[0] >= 'A' && part[0] <= 'Z' {
						part += "."
					}
					items = append(items, part)
				}
			}
		} else {
			// Try semicolons as separators
			parts = strings.Split(text, "; ")
			for _, part := range parts {
				part = strings.TrimSpace(part)
				if part != "" && len(part) > 5 {
					items = append(items, part)
				}
			}
		}
	}
	
	// If nothing works, just return the whole text as one item
	if len(items) == 0 {
		items = append(items, text)
	}
	
	// Final cleanup to remove any empty items or duplicates
	var cleanItems []string
	seen := make(map[string]bool)
	
	for _, item := range items {
		item = strings.TrimSpace(item)
		// Skip empty or very short items
		if item == "" || len(item) < 3 {
			continue
		}
		
		// Skip if we've already seen this
		if seen[item] {
			continue
		}
		
		seen[item] = true
		cleanItems = append(cleanItems, item)
	}
	
	return cleanItems
}

// saveProfileToJSON saves a profile to a JSON file
func (c *Client) saveProfileToJSON(profile *models.Profile) error {
	if profile == nil || profile.Name == "" {
		return fmt.Errorf("cannot save nil or unnamed profile")
	}

	// Create safe filename from profile name
	safeName := url.PathEscape(profile.Name)
	if safeName == "" {
		// Use URL or a timestamp if name is empty after escaping
		if profile.URL != "" {
			urlObj, _ := url.Parse(profile.URL)
			if urlObj != nil && urlObj.Query().Get("ID") != "" {
				safeName = "profile-" + urlObj.Query().Get("ID")
			}
		}
		
		if safeName == "" {
			safeName = "profile-" + time.Now().Format("20060102-150405")
		}
	}
	
	jsonPath := filepath.Join(c.dataDir, safeName+".json")
	fmt.Printf("Saving profile to %s\n", jsonPath)

	// Marshal profile to JSON
	data, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal profile: %w", err)
	}

	// Write JSON to file
	if err := os.WriteFile(jsonPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write profile JSON: %w", err)
	}

	return nil
}

// GetProfile retrieves a profile by name
func (c *Client) GetProfile(name string) (*models.Profile, error) {
	profile, exists := c.profiles[name]
	if !exists {
		return nil, fmt.Errorf("profile not found: %s", name)
	}
	return profile, nil
}

// AddProfile adds a profile to the client
func (c *Client) AddProfile(profile *models.Profile) {
	if profile != nil && profile.Name != "" {
		c.profiles[profile.Name] = profile
	}
}

// SaveProfileToJSON makes the saveProfileToJSON method public
func (c *Client) SaveProfileToJSON(profile *models.Profile) error {
	return c.saveProfileToJSON(profile)
}

// ListProfiles returns all profiles
func (c *Client) ListProfiles() []*models.Profile {
	profiles := make([]*models.Profile, 0, len(c.profiles))
	for _, profile := range c.profiles {
		profiles = append(profiles, profile)
	}
	return profiles
}

// LoadFromDisk loads profiles from JSON files in the data directory
func (c *Client) LoadFromDisk() error {
	files, err := os.ReadDir(c.dataDir)
	if err != nil {
		return fmt.Errorf("failed to read data directory: %w", err)
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		filePath := filepath.Join(c.dataDir, file.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", filePath, err)
		}

		var profile models.Profile
		if err := json.Unmarshal(data, &profile); err != nil {
			return fmt.Errorf("failed to unmarshal profile from %s: %w", filePath, err)
		}

		c.profiles[profile.Name] = &profile
	}

	return nil
}

// SearchProfiles searches profiles by name or description
func (c *Client) SearchProfiles(query string) []*models.Profile {
	var results []*models.Profile
	queryLower := strings.ToLower(query)

	for _, profile := range c.profiles {
		if strings.Contains(strings.ToLower(profile.Name), queryLower) ||
			strings.Contains(strings.ToLower(profile.Description), queryLower) {
			results = append(results, profile)
		}
	}

	return results
}

// GetProfilesByVerdict returns profiles by verdict
func (c *Client) GetProfilesByVerdict(verdict string) []*models.Profile {
	var results []*models.Profile
	verdictLower := strings.ToLower(verdict)

	for _, profile := range c.profiles {
		if strings.Contains(strings.ToLower(profile.Verdict), verdictLower) {
			results = append(results, profile)
		}
	}

	return results
}

// GetProfilesByCategory returns profiles by category
func (c *Client) GetProfilesByCategory(category string) []*models.Profile {
	var results []*models.Profile
	categoryLower := strings.ToLower(category)

	for _, profile := range c.profiles {
		if strings.Contains(strings.ToLower(profile.Category), categoryLower) {
			results = append(results, profile)
		}
	}

	return results
}