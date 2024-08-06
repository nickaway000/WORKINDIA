package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

var db *sql.DB
var jwtKey = []byte(os.Getenv("JWT_SECRET"))

func main() {
    var err error

    err = godotenv.Load(".env")
    if err != nil {
        log.Fatalf("Error loading .env file: %v", err)
    }

    err2 := InitDB()
    if err2 != nil {
        log.Fatalf("Error initializing database: %v", err2)
    }

    http.HandleFunc("/api/signup", RegisterHandler)
    http.HandleFunc("/api/login", LoginHandler)
    http.HandleFunc("/api/shorts/create", CreateShortHandler)
    http.HandleFunc("/api/shorts/feed", FeedHandler)

    fmt.Printf("Starting server at port 9000\n")
    log.Fatal(http.ListenAndServe(":9000", nil))
}

func InitDB() error {
    connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
        os.Getenv("DB_HOST"),
        os.Getenv("DB_PORT"),
        os.Getenv("DB_USER"),
        os.Getenv("DB_PASSWORD"),
        os.Getenv("DB_NAME"))

    var err error
    db, err = sql.Open("postgres", connStr)
    if err != nil {
        return fmt.Errorf("error connecting to the database: %w", err)
    }

    errPing := db.Ping()
    if errPing != nil {
        return fmt.Errorf("error pinging the database: %w", errPing)
    }

    log.Println("Successfully connected to the database")
    return nil
}

func generateJWT(userID int, role string) (string, error) {
    claims := &jwt.RegisteredClaims{
        ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
        Issuer:    fmt.Sprintf("%d", userID),
        Subject:   role,
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    tokenString, err := token.SignedString(jwtKey)
    if err != nil {
        return "", err
    }

    return tokenString, nil
}


func hashPassword(password string) (string, error) {
    bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
    return string(bytes), err
}

func checkPasswordHash(password, hash string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
    return err == nil
}

type RegisterRequest struct {
    Username string `json:"username"`
    Password string `json:"password"`
    Email    string `json:"email"`
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
        return
    }

    var req RegisterRequest
    decoder := json.NewDecoder(r.Body)
    err := decoder.Decode(&req)
    if err != nil {
        http.Error(w, "Invalid request payload", http.StatusBadRequest)
        return
    }

    email := req.Email
    password := req.Password
    username := req.Username

    var exists bool
    err = db.QueryRow("SELECT EXISTS (SELECT 1 FROM users WHERE email=$1)", email).Scan(&exists)
    if err != nil {
        http.Error(w, "Server error", http.StatusInternalServerError)
        return
    }
    if exists {
        http.Error(w, "Email already exists", http.StatusConflict)
        return
    }

    hashedPassword, err := hashPassword(password)
    if err != nil {
        http.Error(w, "Server error", http.StatusInternalServerError)
        return
    }

    var id int
    err = db.QueryRow("INSERT INTO users (username, email, password, role) VALUES ($1, $2, $3, 'user') RETURNING id", username, email, hashedPassword).Scan(&id)
    if err != nil {
        http.Error(w, "Server error", http.StatusInternalServerError)
        return
    }

    response := map[string]interface{}{
        "status":      "Account successfully created",
        "status_code": 200,
        "user_id":     fmt.Sprintf("%d", id),
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
    log.Printf("Account successfully created with userID: %d", id)
}


type LoginRequest struct {
    Username string `json:"username"`
    Password string `json:"password"`
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
        return
    }

    var req LoginRequest
    decoder := json.NewDecoder(r.Body)
    err := decoder.Decode(&req)
    if err != nil {
        http.Error(w, "Invalid request payload", http.StatusBadRequest)
        return
    }

    username := req.Username
    password := req.Password

    var userID int
    var dbPassword, role string
    err = db.QueryRow("SELECT id, password, role FROM users WHERE username=$1", username).Scan(&userID, &dbPassword, &role)
    if err != nil {
        if err == sql.ErrNoRows {
            response := map[string]interface{}{
                "status":      "Incorrect username/password provided. Please retry",
                "status_code": 401,
            }
            w.Header().Set("Content-Type", "application/json")
            json.NewEncoder(w).Encode(response)
        } else {
            http.Error(w, "Server error", http.StatusInternalServerError)
        }
        return
    }

    if !checkPasswordHash(password, dbPassword) {
        response := map[string]interface{}{
            "status":      "Incorrect username/password provided. Please retry",
            "status_code": 401,
        }
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(response)
        return
    }

    token, err := generateJWT(userID, role)
    if err != nil {
        http.Error(w, "Server error", http.StatusInternalServerError)
        return
    }

    response := map[string]interface{}{
        "status":        "Login successful",
        "status_code":   200,
        "user_id":       fmt.Sprintf("%d", userID),
        "role":          role,
        "access_token":  token,
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
    log.Printf("Login successful for userID: %d", userID)
}




type Short struct {
    Category            string `json:"category"`
    Title               string `json:"title"`
    Author              string `json:"author"`
    PublishDate         string `json:"publish_date"`
    Content             string `json:"content"`
    ActualContentLink   string `json:"actual_content_link"`
    Image               string `json:"image"`
    Votes               Votes  `json:"votes"`
}

type Votes struct {
    Upvote   int `json:"upvote"`
    Downvote int `json:"downvote"`
}



func CreateShortHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
        return
    }

    var req Short
    decoder := json.NewDecoder(r.Body)
    err := decoder.Decode(&req)
    if err != nil {
        http.Error(w, "Invalid request payload", http.StatusBadRequest)
        return
    }

    publishDate, err := time.Parse(time.RFC3339, req.PublishDate)
    if err != nil {
        http.Error(w, "Invalid publish date format", http.StatusBadRequest)
        return
    }


    var shortID int
    err = db.QueryRow(`
        INSERT INTO shorts (category, title, author, publish_date, content, actual_content_link, image, upvote, downvote)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) 
        RETURNING id`,
        req.Category, req.Title, req.Author, publishDate, req.Content, req.ActualContentLink, req.Image, req.Votes.Upvote, req.Votes.Downvote).Scan(&shortID)
    
    if err != nil {
        http.Error(w, "Server error", http.StatusInternalServerError)
        return
    }

    response := map[string]interface{}{
        "message":    "Short added successfully",
        "short_id":   fmt.Sprintf("%d", shortID),
        "status_code": 200,
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
    log.Printf("Short created with ID: %d", shortID)
}

func FeedHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
        return
    }

    query := `
        SELECT category, title, author, publish_date, content, actual_content_link, image, upvote, downvote
        FROM shorts
        ORDER BY publish_date DESC, upvote DESC
    `

    log.Println("Executing query:", query)  

    rows, err := db.Query(query)
    if err != nil {
        log.Printf("Error executing query: %v", err)
        http.Error(w, "Server error", http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    var shorts []Short
    for rows.Next() {
        var s Short
        var upvote, downvote int
        var publishDate time.Time

        err := rows.Scan(&s.Category, &s.Title, &s.Author, &publishDate, &s.Content, &s.ActualContentLink, &s.Image, &upvote, &downvote)
        if err != nil {
            log.Printf("Error scanning row: %v", err) 
            http.Error(w, "Server error", http.StatusInternalServerError)
            return
        }

        s.PublishDate = publishDate.Format(time.RFC3339)
        s.Votes = Votes{Upvote: upvote, Downvote: downvote}
        shorts = append(shorts, s)
    }
    
    if err := rows.Err(); err != nil {
        log.Printf("Row iteration error: %v", err)  
        http.Error(w, "Server error", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(shorts)
}

