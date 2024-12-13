package main

import (
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

func main() {
	r := chi.NewRouter()

	r.Use(middleware.RealIP)
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(15 * time.Second))

	hub := NewHub()

	r.HandleFunc("/ws", hub.websocketHandler)

	log.Println("Server is listening on port 8080...")

	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatal(err)
	}
}
