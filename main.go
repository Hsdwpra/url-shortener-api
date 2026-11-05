package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type shortenRequest struct {
	URL string `json:"url"`
}

type shortenResponse struct {
	Code      string `json:"code"`
	ShortURL  string `json:"short_url"`
	LongURL   string `json:"long_url"`
	CreatedAt string `json:"created_at"`
}

type statsResponse struct {
	Code  string  `json:"code"`
	Total int64   `json:"total_clicks"`
	Clicks []Click `json:"recent_clicks"`
}

var baseURL string

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	baseURL = os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:" + port
	}

	store, err := NewStore("data/shorty.db")
	if err != nil {
		log.Fatal(err)
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(10 * time.Second))

	r.Post("/api/shorten", shortenHandler(store))
	r.Get("/{code}", redirectHandler(store))
	r.Get("/api/{code}/stats", statsHandler(store))

	log.Printf("listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

func shortenHandler(s *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req shortenRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		req.URL = strings.TrimSpace(req.URL)
		if !isHTTPURL(req.URL) {
			httpError(w, http.StatusBadRequest, "url must be a valid http(s) URL")
			return
		}
		code, err := s.CreateURL(req.URL)
		if err != nil {
			httpError(w, http.StatusInternalServerError, "failed to create short URL")
			return
		}
		writeJSON(w, http.StatusCreated, shortenResponse{
			Code:      code,
			ShortURL:  baseURL + "/" + code,
			LongURL:   req.URL,
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
		})
	}
}

func redirectHandler(s *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := chi.URLParam(r, "code")
		longURL, err := s.GetURL(code)
		if errors.Is(err, sql.ErrNoRows) {
			httpError(w, http.StatusNotFound, "short URL not found")
			return
		}
		if err != nil {
			httpError(w, http.StatusInternalServerError, "database error")
			return
		}
		go func() {
			if err := s.RecordClick(code, r.Header.Get("Referer"), r.UserAgent()); err != nil {
				log.Printf("record click: %v", err)
			}
		}()
		http.Redirect(w, r, longURL, http.StatusFound)
	}
}

func statsHandler(s *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := chi.URLParam(r, "code")
		if _, err := s.GetURL(code); errors.Is(err, sql.ErrNoRows) {
			httpError(w, http.StatusNotFound, "short URL not found")
			return
		} else if err != nil {
			httpError(w, http.StatusInternalServerError, "database error")
			return
		}
		total, clicks, err := s.Stats(code)
		if err != nil {
			httpError(w, http.StatusInternalServerError, "database error")
			return
		}
		writeJSON(w, http.StatusOK, statsResponse{Code: code, Total: total, Clicks: clicks})
	}
}

func isHTTPURL(s string) bool {
	return len(s) > 8 &&
		(strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")) &&
		strings.Contains(s[8:], ".")
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func httpError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
