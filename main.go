package main

import (
    "log"
    "net/http"
    "os"

    "github.com/Jisin0/autofilterbot/cmd"
)

// Render web services require an HTTP listener. The bot still receives
// Telegram updates using long polling.
func main() {
    port := os.Getenv("PORT")
    if port != "" {
        mux := http.NewServeMux()
        mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
            w.Header().Set("Content-Type", "text/plain; charset=utf-8")
            w.WriteHeader(http.StatusOK)
            _, _ = w.Write([]byte("Telegram bot process is running"))
        })
        go func() {
            log.Printf("Starting HTTP listener on port %s", port)
            if err := http.ListenAndServe(":"+port, mux); err != nil {
                log.Printf("Health listener stopped: %v", err)
            }
        }()
    }
    cmd.Execute()
}
