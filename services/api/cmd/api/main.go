package main

import (
    "log"

    "github.com/example/cs2trader/internal/server"
)

func main() {
    apiServer, err := server.NewServerFromEnv()
    if err != nil {
        log.Fatalf("failed to initialize server: %v", err)
    }

    if err := apiServer.Start(); err != nil {
        log.Fatalf("server exited with error: %v", err)
    }
}

