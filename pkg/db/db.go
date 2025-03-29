package db

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
	"github.com/marcusziade/jewornotjew/pkg/models"
)

// DB represents the database connection
type DB struct {
	db *sql.DB
}

// New creates a new database connection
func New(dbPath string) (*DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{db: db}, nil
}

// InitSchema initializes the database schema
func (d *DB) InitSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS profiles (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		url TEXT NOT NULL,
		verdict TEXT NOT NULL,
		description TEXT,
		score REAL,
		category TEXT,
		image_url TEXT,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS pros (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		profile_id INTEGER NOT NULL,
		text TEXT NOT NULL,
		FOREIGN KEY (profile_id) REFERENCES profiles (id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS cons (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		profile_id INTEGER NOT NULL,
		text TEXT NOT NULL,
		FOREIGN KEY (profile_id) REFERENCES profiles (id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_profiles_name ON profiles (name);
	CREATE INDEX IF NOT EXISTS idx_profiles_verdict ON profiles (verdict);
	CREATE INDEX IF NOT EXISTS idx_profiles_category ON profiles (category);
	`

	_, err := d.db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	return nil
}

// InsertProfile inserts a profile into the database
func (d *DB) InsertProfile(profile *models.Profile) error {
	tx, err := d.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert profile
	stmt, err := tx.Prepare(`
		INSERT INTO profiles (name, url, verdict, description, score, category, image_url, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(name) DO UPDATE SET
			url = excluded.url,
			verdict = excluded.verdict,
			description = excluded.description,
			score = excluded.score,
			category = excluded.category,
			image_url = excluded.image_url,
			updated_at = excluded.updated_at
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare profile statement: %w", err)
	}
	defer stmt.Close()

	res, err := stmt.Exec(
		profile.Name,
		profile.URL,
		profile.Verdict,
		profile.Description,
		profile.Score,
		profile.Category,
		profile.ImageURL,
		profile.CreatedAt,
		profile.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert profile: %w", err)
	}

	// Get profile ID
	var profileID int64
	// Check if profile already exists by name
	var exists bool
	if err := tx.QueryRow("SELECT EXISTS(SELECT 1 FROM profiles WHERE name = ?)", profile.Name).Scan(&exists); err != nil {
		return fmt.Errorf("failed to check if profile exists: %w", err)
	}
	
	if exists {
		// Profile already exists, get its ID
		row := tx.QueryRow("SELECT id FROM profiles WHERE name = ?", profile.Name)
		if err := row.Scan(&profileID); err != nil {
			return fmt.Errorf("failed to get existing profile ID: %w", err)
		}

		// Delete existing pros and cons
		if _, err := tx.Exec("DELETE FROM pros WHERE profile_id = ?", profileID); err != nil {
			return fmt.Errorf("failed to delete existing pros: %w", err)
		}
		if _, err := tx.Exec("DELETE FROM cons WHERE profile_id = ?", profileID); err != nil {
			return fmt.Errorf("failed to delete existing cons: %w", err)
		}
	} else {
		// Get the ID of the newly inserted row
		profileID, err = res.LastInsertId()
		if err != nil {
			return fmt.Errorf("failed to get profile ID: %w", err)
		}
	}

	// Insert pros
	if len(profile.Pros) > 0 {
		prosStmt, err := tx.Prepare("INSERT INTO pros (profile_id, text) VALUES (?, ?)")
		if err != nil {
			return fmt.Errorf("failed to prepare pros statement: %w", err)
		}
		defer prosStmt.Close()

		for _, pro := range profile.Pros {
			if _, err := prosStmt.Exec(profileID, pro); err != nil {
				return fmt.Errorf("failed to insert pro: %w", err)
			}
		}
	}

	// Insert cons
	if len(profile.Cons) > 0 {
		consStmt, err := tx.Prepare("INSERT INTO cons (profile_id, text) VALUES (?, ?)")
		if err != nil {
			return fmt.Errorf("failed to prepare cons statement: %w", err)
		}
		defer consStmt.Close()

		for _, con := range profile.Cons {
			if _, err := consStmt.Exec(profileID, con); err != nil {
				return fmt.Errorf("failed to insert con: %w", err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetProfile retrieves a profile by name
func (d *DB) GetProfile(name string) (*models.Profile, error) {
	profile := &models.Profile{}

	// Get profile data
	row := d.db.QueryRow(`
		SELECT name, url, verdict, description, score, category, image_url, created_at, updated_at 
		FROM profiles 
		WHERE name = ?
	`, name)

	err := row.Scan(
		&profile.Name,
		&profile.URL,
		&profile.Verdict,
		&profile.Description,
		&profile.Score,
		&profile.Category,
		&profile.ImageURL,
		&profile.CreatedAt,
		&profile.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("profile not found: %s", name)
		}
		return nil, fmt.Errorf("failed to scan profile: %w", err)
	}

	// Get profile ID
	var profileID int
	err = d.db.QueryRow("SELECT id FROM profiles WHERE name = ?", name).Scan(&profileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get profile ID: %w", err)
	}

	// Get pros
	prosRows, err := d.db.Query("SELECT text FROM pros WHERE profile_id = ?", profileID)
	if err != nil {
		return nil, fmt.Errorf("failed to query pros: %w", err)
	}
	defer prosRows.Close()

	for prosRows.Next() {
		var pro string
		if err := prosRows.Scan(&pro); err != nil {
			return nil, fmt.Errorf("failed to scan pro: %w", err)
		}
		profile.Pros = append(profile.Pros, pro)
	}

	// Get cons
	consRows, err := d.db.Query("SELECT text FROM cons WHERE profile_id = ?", profileID)
	if err != nil {
		return nil, fmt.Errorf("failed to query cons: %w", err)
	}
	defer consRows.Close()

	for consRows.Next() {
		var con string
		if err := consRows.Scan(&con); err != nil {
			return nil, fmt.Errorf("failed to scan con: %w", err)
		}
		profile.Cons = append(profile.Cons, con)
	}

	return profile, nil
}

// ListProfiles returns all profiles
func (d *DB) ListProfiles() ([]*models.Profile, error) {
	// Get all profile IDs
	rows, err := d.db.Query(`
		SELECT id, name, url, verdict, description, score, category, image_url, created_at, updated_at 
		FROM profiles
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query profiles: %w", err)
	}
	defer rows.Close()

	profiles := []*models.Profile{}
	profileIDs := map[int64]*models.Profile{}

	for rows.Next() {
		profile := &models.Profile{}
		var id int64
		err := rows.Scan(
			&id,
			&profile.Name,
			&profile.URL,
			&profile.Verdict,
			&profile.Description,
			&profile.Score,
			&profile.Category,
			&profile.ImageURL,
			&profile.CreatedAt,
			&profile.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan profile: %w", err)
		}

		profiles = append(profiles, profile)
		profileIDs[id] = profile
	}

	// Get all pros
	prosRows, err := d.db.Query("SELECT profile_id, text FROM pros")
	if err != nil {
		return nil, fmt.Errorf("failed to query pros: %w", err)
	}
	defer prosRows.Close()

	for prosRows.Next() {
		var profileID int64
		var pro string
		if err := prosRows.Scan(&profileID, &pro); err != nil {
			return nil, fmt.Errorf("failed to scan pro: %w", err)
		}
		if profile, ok := profileIDs[profileID]; ok {
			profile.Pros = append(profile.Pros, pro)
		}
	}

	// Get all cons
	consRows, err := d.db.Query("SELECT profile_id, text FROM cons")
	if err != nil {
		return nil, fmt.Errorf("failed to query cons: %w", err)
	}
	defer consRows.Close()

	for consRows.Next() {
		var profileID int64
		var con string
		if err := consRows.Scan(&profileID, &con); err != nil {
			return nil, fmt.Errorf("failed to scan con: %w", err)
		}
		if profile, ok := profileIDs[profileID]; ok {
			profile.Cons = append(profile.Cons, con)
		}
	}

	return profiles, nil
}

// SearchProfiles searches profiles by name, verdict, or description
func (d *DB) SearchProfiles(query string) ([]*models.Profile, error) {
	queryPattern := "%" + query + "%"
	
	// Get matching profile IDs
	rows, err := d.db.Query(`
		SELECT id, name, url, verdict, description, score, category, image_url, created_at, updated_at 
		FROM profiles
		WHERE name LIKE ? OR verdict LIKE ? OR description LIKE ?
	`, queryPattern, queryPattern, queryPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to query profiles: %w", err)
	}
	defer rows.Close()

	profiles := []*models.Profile{}
	profileIDs := map[int64]*models.Profile{}

	for rows.Next() {
		profile := &models.Profile{}
		var id int64
		err := rows.Scan(
			&id,
			&profile.Name,
			&profile.URL,
			&profile.Verdict,
			&profile.Description,
			&profile.Score,
			&profile.Category,
			&profile.ImageURL,
			&profile.CreatedAt,
			&profile.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan profile: %w", err)
		}

		profiles = append(profiles, profile)
		profileIDs[id] = profile
	}

	// Get pros for matching profiles
	for id, profile := range profileIDs {
		prosRows, err := d.db.Query("SELECT text FROM pros WHERE profile_id = ?", id)
		if err != nil {
			return nil, fmt.Errorf("failed to query pros: %w", err)
		}

		for prosRows.Next() {
			var pro string
			if err := prosRows.Scan(&pro); err != nil {
				prosRows.Close()
				return nil, fmt.Errorf("failed to scan pro: %w", err)
			}
			profile.Pros = append(profile.Pros, pro)
		}
		prosRows.Close()

		// Get cons for matching profiles
		consRows, err := d.db.Query("SELECT text FROM cons WHERE profile_id = ?", id)
		if err != nil {
			return nil, fmt.Errorf("failed to query cons: %w", err)
		}

		for consRows.Next() {
			var con string
			if err := consRows.Scan(&con); err != nil {
				consRows.Close()
				return nil, fmt.Errorf("failed to scan con: %w", err)
			}
			profile.Cons = append(profile.Cons, con)
		}
		consRows.Close()
	}

	return profiles, nil
}

// Close closes the database connection
func (d *DB) Close() error {
	return d.db.Close()
}