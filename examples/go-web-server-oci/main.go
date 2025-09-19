package main

import (
    "encoding/json"
    "fmt"
    "io"
    "log"
    "net/http"
    "os"
    "time"
)

type echoResponse struct {
    Timestamp string              `json:"timestamp"`
    Method    string              `json:"method"`
    Path      string              `json:"path"`
    Headers   map[string][]string `json:"headers"`
    Body      string              `json:"body"`
}

func echoHandler(w http.ResponseWriter, r *http.Request) {
    defer r.Body.Close()
    body, _ := io.ReadAll(r.Body)

    resp := echoResponse{
        Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
        Method:    r.Method,
        Path:      r.URL.Path,
        Headers:   r.Header,
        Body:      string(body),
    }

    w.Header().Set("Content-Type", "application/json")
    enc := json.NewEncoder(w)
    enc.SetIndent("", "  ")
    _ = enc.Encode(resp)
}

func main() {
    mux := http.NewServeMux()
    mux.HandleFunc("/", echoHandler)

    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }
    addr := fmt.Sprintf(":%s", port)
    log.Printf("Echo server listening on %s", addr)
    if err := http.ListenAndServe(addr, mux); err != nil {
        log.Fatalf("server error: %v", err)
    }
}

