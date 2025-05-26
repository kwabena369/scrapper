package main

import (
    "log"
    "net/http"
    "os"

    "github.com/gorilla/mux"
    "github.com/joho/godotenv"
    "github.com/kwabena369/scrapper/internal/db"
    "github.com/kwabena369/scrapper/internal/handlers"
)

func main() {
    err := godotenv.Load()
    if err != nil {
        log.Fatal("Error loading .env file")
    }
    port := os.Getenv("PORT")
    if port == "" {
        port = "8000"
    }

    // Connect to MongoDB
    db.ConnectMongo()
    defer db.DisconnectMongo()

    router := mux.NewRouter()
    router.Use(handlers.TheLoggingMiddleware)
    routerV1 := router.PathPrefix("/v1").Subrouter()

    // User CRUD endpoints with MongoDB client
    client := db.Client
    routerV1.HandleFunc("/users", handlers.CreateUser(client)).Methods("POST")
    routerV1.HandleFunc("/users/{id}", handlers.GetUser(client)).Methods("GET")
    routerV1.HandleFunc("/users/{id}", handlers.UpdateUser(client)).Methods("PUT")
    routerV1.HandleFunc("/users/{id}", handlers.DeleteUser(client)).Methods("DELETE")

    log.Printf("Starting server on port %s", port)
    if err := http.ListenAndServe(":"+port, router); err != nil {
        log.Fatalf("Could not start server: %s\n", err.Error())
    }
}