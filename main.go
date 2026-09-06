package main

import (
	"log"
	"net/http"
	"os"

	"featureflags/internal/handlers"
	"featureflags/internal/middleware"
	"featureflags/internal/store"
)

func main() {
	s := store.NewStore()

	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", handlers.Health())
	mux.HandleFunc("POST /flags", handlers.CreateFlag(s))
	mux.HandleFunc("GET /flags", handlers.ListFlags(s))
	mux.HandleFunc("GET /flags/{key}", handlers.GetFlag(s))
	mux.HandleFunc("PUT /flags/{key}", handlers.UpdateFlag(s))
	mux.HandleFunc("DELETE /flags/{key}", handlers.DeleteFlag(s))
	mux.HandleFunc("GET /flags/{key}/evaluate", handlers.EvaluateFlag(s))

	handler := middleware.Logging(mux)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("listening on :%s", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
