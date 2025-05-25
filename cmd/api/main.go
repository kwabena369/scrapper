package main

import (
    "log"
    "net/http"
    "os"

    "github.com/gorilla/mux"
    "github.com/joho/godotenv"
    "github.com/kwabena369/scrapper/handlers"
)

func healthCheck(w http.ResponseWriter, r *http.Request) {
    handlers.RespondWithJSON(w, 200, map[string]string{"message": "the server is up and running "}) // Capitalized
}

func whatPower(w http.ResponseWriter, r *http.Request) {
    handlers.RespondWithJSON(w, 200, map[string]string{"message": "the power is in the light"}) // Capitalized
}

func main() {
    err := godotenv.Load()
    if err != nil {
        log.Fatal("Error loading .env file")
    }
    port := os.Getenv("PORT")
    if port == "" {
        port = "8000"
    }
    router := mux.NewRouter()
    router.Use(handlers.TheLoggingMiddleware)
    routerV1 := router.PathPrefix("/v1").Subrouter()
    routerV1.HandleFunc("/what", whatPower).Methods("GET")
    router.HandleFunc("/", healthCheck).Methods("GET")
    log.Printf("Starting server on port %s", port)
    if err := http.ListenAndServe(":"+port, router); err != nil {
        log.Fatalf("Could not start server: %s\n", err.Error())
    }
}