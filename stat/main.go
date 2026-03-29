package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type Post struct {
	ID        int    `json:"id"`
	Platform  string `json:"platform"`
	Author    string `json:"author"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

type StatsRequest struct {
	IncludeKeywords []string `json:"include_keywords"`
	ExcludeKeywords []string `json:"exclude_keywords"`
	FromDate        string   `json:"from_date"`
	ToDate          string   `json:"to_date"`
	ExampleLimit    int      `json:"example_limit"`
	PostLimit       int      `json:"post_limit"`
}

type KeywordCount struct {
	Keyword string `json:"keyword"`
	Count   int    `json:"count"`
}

type TrendPoint struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

type StatsResponse struct {
	MentionCount int            `json:"mention_count"`
	TopKeywords  []KeywordCount `json:"top_keywords"`
	Trends       []TrendPoint   `json:"trends"`
	ExamplePosts []Post         `json:"example_posts"`
	Posts        []Post         `json:"posts"`
}

var stopWords = map[string]bool{
	"the": true, "a": true, "an": true, "and": true, "or": true, "to": true,
	"of": true, "in": true, "on": true, "for": true, "with": true, "is": true,
	"are": true, "it": true, "this": true, "that": true, "my": true, "our": true,
	"your": true, "but": true, "from": true, "at": true, "was": true,
}

func parseDateRange(fromDate, toDate string) (string, string, error) {
	layout := "2006-01-02"
	now := time.Now()

	if fromDate == "" {
		fromDate = now.AddDate(0, 0, -30).Format(layout)
	}
	if toDate == "" {
		toDate = now.Format(layout)
	}

	if _, err := time.Parse(layout, fromDate); err != nil {
		return "", "", fmt.Errorf("invalid from_date: %w", err)
	}
	if _, err := time.Parse(layout, toDate); err != nil {
		return "", "", fmt.Errorf("invalid to_date: %w", err)
	}

	return fromDate, toDate, nil
}

func containsAny(text string, words []string) bool {
	if len(words) == 0 {
		return true
	}
	l := strings.ToLower(text)
	for _, w := range words {
		w = strings.TrimSpace(strings.ToLower(w))
		if w == "" {
			continue
		}
		if strings.Contains(l, w) {
			return true
		}
	}
	return false
}

func containsNone(text string, words []string) bool {
	l := strings.ToLower(text)
	for _, w := range words {
		w = strings.TrimSpace(strings.ToLower(w))
		if w == "" {
			continue
		}
		if strings.Contains(l, w) {
			return false
		}
	}
	return true
}

func extractTopKeywords(posts []Post, include, exclude []string, topN int) []KeywordCount {
	re := regexp.MustCompile(`[a-zA-Z']+`)
	freq := map[string]int{}
	excludedMap := map[string]bool{}

	for _, w := range exclude {
		w = strings.ToLower(strings.TrimSpace(w))
		if w != "" {
			excludedMap[w] = true
		}
	}

	for _, post := range posts {
		tokens := re.FindAllString(strings.ToLower(post.Content), -1)
		for _, t := range tokens {
			if len(t) <= 2 || stopWords[t] || excludedMap[t] {
				continue
			}
			freq[t]++
		}
	}

	items := make([]KeywordCount, 0, len(freq))
	for k, c := range freq {
		items = append(items, KeywordCount{Keyword: k, Count: c})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].Count == items[j].Count {
			return items[i].Keyword < items[j].Keyword
		}
		return items[i].Count > items[j].Count
	})

	if len(items) > topN {
		items = items[:topN]
	}
	return items
}

func buildTrends(posts []Post) []TrendPoint {
	counts := map[string]int{}
	for _, p := range posts {
		if len(p.CreatedAt) >= 10 {
			counts[p.CreatedAt[:10]]++
		}
	}

	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	trends := make([]TrendPoint, 0, len(keys))
	for _, d := range keys {
		trends = append(trends, TrendPoint{Date: d, Count: counts[d]})
	}
	return trends
}

func fetchFilteredPosts(db *sql.DB, req StatsRequest) ([]Post, error) {
	fromDate, toDate, err := parseDateRange(req.FromDate, req.ToDate)
	if err != nil {
		return nil, err
	}

	rows, err := db.Query(
		`SELECT id, platform, author, content, created_at
		 FROM posts
		 WHERE date(created_at) BETWEEN date(?) AND date(?)
		 ORDER BY datetime(created_at) DESC`,
		fromDate,
		toDate,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	posts := []Post{}
	for rows.Next() {
		var p Post
		if err := rows.Scan(&p.ID, &p.Platform, &p.Author, &p.Content, &p.CreatedAt); err != nil {
			return nil, err
		}

		if !containsAny(p.Content, req.IncludeKeywords) {
			continue
		}
		if !containsNone(p.Content, req.ExcludeKeywords) {
			continue
		}
		posts = append(posts, p)
	}

	if req.PostLimit <= 0 {
		req.PostLimit = 500
	}
	if len(posts) > req.PostLimit {
		posts = posts[:req.PostLimit]
	}

	return posts, nil
}

func setupDatabase(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS posts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			platform TEXT NOT NULL,
			author TEXT NOT NULL,
			content TEXT NOT NULL,
			created_at TEXT NOT NULL
		)
	`)
	return err
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func main() {
	port := os.Getenv("STAT_PORT")
	if port == "" {
		port = "8002"
	}
	sqlitePath := os.Getenv("SQLITE_PATH")
	if sqlitePath == "" {
		sqlitePath = "./listenai.db"
	}

	db, err := sql.Open("sqlite", sqlitePath)
	if err != nil {
		log.Fatalf("failed to open sqlite: %v", err)
	}
	defer db.Close()

	if err := setupDatabase(db); err != nil {
		log.Fatalf("failed to setup database: %v", err)
	}

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "stat", "port": port})
	})

	http.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}

		var req StatsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}

		posts, err := fetchFilteredPosts(db, req)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		exampleLimit := req.ExampleLimit
		if exampleLimit <= 0 {
			exampleLimit = 5
		}

		examplePosts := posts
		if len(examplePosts) > exampleLimit {
			examplePosts = examplePosts[:exampleLimit]
		}

		resp := StatsResponse{
			MentionCount: len(posts),
			TopKeywords:  extractTopKeywords(posts, req.IncludeKeywords, req.ExcludeKeywords, 10),
			Trends:       buildTrends(posts),
			ExamplePosts: examplePosts,
			Posts:        posts,
		}
		writeJSON(w, http.StatusOK, resp)
	})

	addr := ":" + port
	log.Printf("stat service listening on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
