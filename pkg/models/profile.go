package models

// Profile represents a person profile from jewornotjew.com
type Profile struct {
	Name        string   `json:"name"`
	URL         string   `json:"url"`
	Verdict     string   `json:"verdict"`
	Description string   `json:"description"`
	Pros        []string `json:"pros"`
	Cons        []string `json:"cons"`
	Score       float64  `json:"score"`
	Category    string   `json:"category"`
	ImageURL    string   `json:"image_url"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
}