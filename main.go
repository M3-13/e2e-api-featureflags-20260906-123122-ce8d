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
	mux.Handle("POST /flags", middleware.Auth(handlers.CreateFlag(s)))
	mux.HandleFunc("GET /flags", handlers.ListFlags(s))
	mux.HandleFunc("GET /flags/{key}", handlers.GetFlag(s))
	mux.Handle("PUT /flags/{key}", middleware.Auth(handlers.UpdateFlag(s)))
	mux.Handle("DELETE /flags/{key}", middleware.Auth(handlers.DeleteFlag(s)))
	mux.HandleFunc("GET /flags/{key}/evaluate", handlers.EvaluateFlag(s))

	handler := middleware.Recover(middleware.Logging(mux))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("listening on 127.0.0.1:%s", port)
	if err := http.ListenAndServe("127.0.0.1:"+port, handler); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
