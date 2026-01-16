package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "9090"
	}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("Error reading body: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		log.Printf("\n--- NEW WEBHOOK RECEIVED [%s] ---", time.Now().Format(time.RFC3339))
		log.Printf("Headers: %v", r.Header)
		log.Printf("Body: %s", string(body))

		w.WriteHeader(http.StatusOK)
		if _, err := fmt.Fprintf(w, "Webhook received"); err != nil {
			log.Printf("Failed to write response: %v", err)
		}
	})

	log.Printf("Webhook mock server started on :%s/", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
