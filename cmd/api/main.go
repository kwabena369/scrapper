package main

import (
    "log"
    "net/http"
    "os"

    gorillaHandlers "github.com/gorilla/handlers"
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
        port = "8080"
    }

    db.ConnectMongo()
    db.ConnectFirebase()
    defer db.DisconnectMongo()

    router := mux.NewRouter()
    router.Use(handlers.TheLoggingMiddleware)

    routerV1 := router.PathPrefix("/v1").Subrouter()
    client := db.Client
    routerV1.HandleFunc("/users", handlers.CreateUser(client)).Methods("POST")
    routerV1.HandleFunc("/users/{id}", handlers.GetUser(client)).Methods("GET")
    routerV1.HandleFunc("/users/{id}", handlers.UpdateUser(client)).Methods("PUT")
    routerV1.HandleFunc("/users/{id}", handlers.DeleteUser(client)).Methods("DELETE")

    // Add CORS middleware
    corsHandler := gorillaHandlers.CORS(
        gorillaHandlers.AllowedOrigins([]string{"http://localhost:3002"}), // Frontend origin
        gorillaHandlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
        gorillaHandlers.AllowedHeaders([]string{"Content-Type", "Authorization"}),
        gorillaHandlers.AllowCredentials(),
    )(router)

    log.Printf("Starting server on port %s", port)
    if err := http.ListenAndServe(":"+port, corsHandler); err != nil {
        log.Fatalf("Could not start server: %s\n", err.Error())
    }
}