package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func loadEnv() {
	if os.Getenv("DOCKER_ENV") == "true" {
		godotenv.Load(".env")
	} else {
		godotenv.Load(".env.local")
	}
}
func getEnv(key, fallback string) string {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	return val
}

// STRUCTURE
type Url struct {
	ID   string    `json:"id"`
	OURL string    `json:"original"`
	NURL string    `json:"new_url"`
	DATE time.Time `json:"date"`
}

// HASHING
func Gen_Url(ourl string) string {
	hasher := md5.New()
	hasher.Write([]byte(ourl))
	data := hasher.Sum(nil)
	h := hex.EncodeToString(data)
	return h[:8]
}

// validate URL is non-empty and well-formed
func isValidURL(raw string) bool {
	if strings.TrimSpace(raw) == "" {
		return false
	}
	u, err := url.ParseRequestURI(raw)
	if err != nil {
		return false
	}
	// must have http or https scheme
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	return true
}

func Create_url(ourl string, baseURL string) (string, error) {
	su := Gen_Url(ourl)

	exist, err := Get_url(su)
	if err == nil {
		// return the full short URL, not just the hash
		return baseURL + "/r/" + exist.NURL, nil
	}

	u := Url{
		ID:   su,
		OURL: ourl,
		NURL: su,
		DATE: time.Now(),
	}

	//  use INSERT ON CONFLICT DO NOTHING at DB level
	err = insertLink(u)
	if err != nil {
		return "", err
	}

	// return full short URL
	return baseURL + "/r/" + su, nil
}

func Get_url(id string) (Url, error) {
	return getLink(id)
}

// SERVER
func handler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("URL Shortener API Running 🚀"))
	BaseURL := getEnv("BASE_URL", "http://localhost:3000")
	start := time.Now()
	defer func() {
		log.Println("handler time:", time.Since(start))
	}()
	//  only allow POST
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var data struct {
		URL string `json:"url"`
	}

	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	//validate URL before processing
	if !isValidURL(data.URL) {
		http.Error(w, "invalid or empty URL", http.StatusBadRequest)
		return
	}

	//  Create_url now returns the full short URL
	shurl, err := Create_url(data.URL, BaseURL)
	if err != nil {
		log.Println("Create_url error:", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	response := struct {
		Shurl string `json:"surl"`
	}{Shurl: shurl}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// REDIRECT FUNCTION
func redirect(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/r/")

	//  reject empty id
	if strings.TrimSpace(id) == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}

	// redis cache check
	start := time.Now()
	url, err := getcachedUrl(id)
	log.Println("redis:", time.Since(start))
	if err == nil {
		log.Println("CACHE HIT")
		http.Redirect(w, r, url, http.StatusFound)
		return
	}
	log.Println("CACHE MISS")
	// cache miss — fallback to DB
	u, err := Get_url(id)
	if err != nil {
		http.Error(w, "URL not found", http.StatusNotFound)
		return
	}

	_ = cacheURL(id, u.OURL)
	http.Redirect(w, r, u.OURL, http.StatusFound)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent) // explicitly return 204
			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	loadEnv()
	InitDB()

	// handle initRedis failure
	if err := initRedis(); err != nil {
		log.Fatal("Redis init failed:", err)
	}
	baseURL := getEnv("BASE_URL", "http://localhost:3000")
	port := getEnv("PORT", "3000")

	mux := http.NewServeMux()

	mux.HandleFunc("/urls", handler)
	mux.HandleFunc("/r/", redirect)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	log.Println("Server running on", baseURL)

	err := http.ListenAndServe(":"+port, corsMiddleware(mux))
	if err != nil {
		log.Fatal(err)
	}

}
