package main

import (
    "log"
    "os"

    "weather/internal/db"
    "weather/internal/server"
)

func main() {
    dsn := os.Getenv("DATABASE_URL")
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    dbConn, err := db.Connect(dsn)
    if err != nil {
        log.Fatalf("failed to connect to db: %v", err)
    }
    defer dbConn.Close()

    if err := db.Migrate(dsn, "migrations"); err != nil {
        log.Fatalf("failed to migrate: %v", err)
    }

    srv := server.New(dbConn)
    srv.Addr = "0.0.0.0:" + port
    log.Printf("starting server on %s", srv.Addr)
    if err := srv.ListenAndServe(); err != nil {
        log.Fatalf("server failed: %v", err)
    }
}