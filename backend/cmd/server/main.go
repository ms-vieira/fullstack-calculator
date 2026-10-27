package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/ms-vieira/fullstack-calculator/backend/internal/httpapi"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	value, err := strconv.Atoi(port)
	if err != nil || value < 1 || value > 65535 {
		return fmt.Errorf("PORT must be an integer between 1 and 65535")
	}
	handler := httpapi.NewHandler()
	if directory := os.Getenv("STATIC_DIR"); directory != "" {
		index, err := os.Stat(filepath.Join(directory, "index.html"))
		if err != nil || !index.Mode().IsRegular() {
			return fmt.Errorf("STATIC_DIR must contain a regular index.html file")
		}
		handler = httpapi.NewWebHandler(os.DirFS(directory))
	}
	server := &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
		ReadHeaderTimeout: 2 * time.Second,
		ReadTimeout:       3 * time.Second,
		WriteTimeout:      3 * time.Second,
		IdleTimeout:       30 * time.Second,
	}
	result := make(chan error, 1)
	go func() { result <- server.ListenAndServe() }()
	log.Printf("Calculator listening on http://localhost:%s", port)
	select {
	case err := <-result:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			_ = server.Close()
			return fmt.Errorf("shut down server: %w", err)
		}
		return nil
	}
}
