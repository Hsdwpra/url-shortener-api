package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// Store wraps the SQLite database.
type Store struct {
	db *sql.DB
}

func NewStore(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, err
	}
	schema := `
	CREATE TABLE IF NOT EXISTS urls (
		code       TEXT PRIMARY KEY,
		long_url   TEXT NOT NULL,
		created_at TEXT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS clicks (
		id      INTEGER PRIMARY KEY AUTOINCREMENT,
		code    TEXT NOT NULL REFERENCES urls(code),
		ts      TEXT NOT NULL,
		referrer TEXT NOT NULL DEFAULT '',
		user_agent TEXT NOT NULL DEFAULT ''
	);
	CREATE INDEX IF NOT EXISTS idx_clicks_code ON clicks(code);
	`
	if _, err := db.Exec(schema); err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

// CreateURL stores a long URL under a new random code.
func (s *Store) CreateURL(longURL string) (string, error) {
	code, err := newCode()
	if err != nil {
		return "", err
	}
	_, err = s.db.Exec(`INSERT INTO urls (code, long_url, created_at) VALUES (?, ?, ?)`,
		code, longURL, time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return "", err
	}
	return code, nil
}

func (s *Store) GetURL(code string) (string, error) {
	var longURL string
	err := s.db.QueryRow(`SELECT long_url FROM urls WHERE code = ?`, code).Scan(&longURL)
	return longURL, err
}

// RecordClick logs one redirect hit.
func (s *Store) RecordClick(code, referrer, userAgent string) error {
	_, err := s.db.Exec(`INSERT INTO clicks (code, ts, referrer, user_agent) VALUES (?, ?, ?, ?)`,
		code, time.Now().UTC().Format(time.RFC3339), referrer, userAgent)
	return err
}

// Stats returns click history for a code.
type Click struct {
	Timestamp string `json:"timestamp"`
	Referrer  string `json:"referrer"`
	UserAgent string `json:"user_agent"`
}

func (s *Store) Stats(code string) (int64, []Click, error) {
	var total int64
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM clicks WHERE code = ?`, code).Scan(&total); err != nil {
		return 0, nil, err
	}
	rows, err := s.db.Query(`SELECT ts, referrer, user_agent FROM clicks WHERE code = ? ORDER BY id DESC LIMIT 50`, code)
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()
	var clicks []Click
	for rows.Next() {
		var c Click
		if err := rows.Scan(&c.Timestamp, &c.Referrer, &c.UserAgent); err != nil {
			return 0, nil, err
		}
		clicks = append(clicks, c)
	}
	return total, clicks, rows.Err()
}

func newCode() (string, error) {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate code: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
