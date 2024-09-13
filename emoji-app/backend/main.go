package main

import (
    "database/sql"
    "encoding/json"
    "log"
    "net/http"
    "os"
    "strconv"

    "github.com/gorilla/mux"
    "github.com/joho/godotenv"
    _ "github.com/lib/pq"
    "github.com/rs/cors"
)

type Emoji struct {
    ID        int    `json:"id"`
    Character string `json:"character"`
}

var db *sql.DB

func main() {
    if err := godotenv.Load(); err != nil {
        log.Fatal("Error loading .env file")
    }

    var err error
    db, err = sql.Open("postgres", os.Getenv("DATABASE_URL"))
    if err != nil {
        log.Fatal("Error connecting to the database:", err)
    }
    defer db.Close()

    log.Println("Successfully connected to the database")

    r := mux.NewRouter()
    r.HandleFunc("/emojis", getEmojis).Methods("GET")
    r.HandleFunc("/emojis", addEmoji).Methods("POST")
    r.HandleFunc("/emojis/{id}", deleteEmoji).Methods("DELETE")

    // CORSミドルウェアを設定
    c := cors.New(cors.Options{
        AllowedOrigins: []string{"http://localhost:3000"},
        AllowedMethods: []string{"GET", "POST", "DELETE", "OPTIONS"},
        AllowedHeaders: []string{"*"},
        AllowCredentials: true,
    })

    // ミドルウェアをルーターに適用
    handler := c.Handler(r)

    log.Println("Starting server on :8080")
    log.Fatal(http.ListenAndServe(":8080", handler))
}

func getEmojis(w http.ResponseWriter, r *http.Request) {
    rows, err := db.Query("SELECT id, character FROM emojis")
    if err != nil {
        log.Println("Error querying database:", err)
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    var emojis []Emoji
    for rows.Next() {
        var emoji Emoji
        if err := rows.Scan(&emoji.ID, &emoji.Character); err != nil {
            log.Println("Error scanning row:", err)
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        emojis = append(emojis, emoji)
    }

    w.Header().Set("Content-Type", "application/json")
    if err := json.NewEncoder(w).Encode(emojis); err != nil {
        log.Println("Error encoding response:", err)
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
}

func addEmoji(w http.ResponseWriter, r *http.Request) {
    var emoji Emoji
    if err := json.NewDecoder(r.Body).Decode(&emoji); err != nil {
        log.Println("Error decoding request body:", err)
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    if _, err := db.Exec("INSERT INTO emojis (character) VALUES ($1)", emoji.Character); err != nil {
        log.Println("Error inserting into database:", err)
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusCreated)
}

func deleteEmoji(w http.ResponseWriter, r *http.Request) {
    // CORS ヘッダーを設定
    w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
    w.Header().Set("Access-Control-Allow-Methods", "DELETE, OPTIONS")
    w.Header().Set("Access-Control-Allow-Headers", "*")

    // OPTIONSリクエスト（プリフライトリクエスト）の場合は、ここで処理を終了
    if r.Method == "OPTIONS" {
        w.WriteHeader(http.StatusOK)
        return
    }

    vars := mux.Vars(r)
    id, err := strconv.Atoi(vars["id"])
    if err != nil {
        log.Println("Error parsing id:", err)
        http.Error(w, "Invalid ID", http.StatusBadRequest)
        return
    }

    result, err := db.Exec("DELETE FROM emojis WHERE id = $1", id)
    if err != nil {
        log.Println("Error deleting from database:", err)
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    rowsAffected, err := result.RowsAffected()
    if err != nil {
        log.Println("Error getting rows affected:", err)
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    if rowsAffected == 0 {
        http.Error(w, "Emoji not found", http.StatusNotFound)
        return
    }

    w.WriteHeader(http.StatusNoContent)
}