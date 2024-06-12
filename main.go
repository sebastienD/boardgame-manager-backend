package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
)

func main() {
	// init logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// init database
	gdb := NewGameDatabase()
	// TODO retry connection
	if err := gdb.Connect(); err != nil {
		slog.Error(fmt.Sprintf("Database connection failed : %v", err), "cause", err.Error())
		os.Exit(1)
	}
	defer gdb.Close()

	// define routes and handlers
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/", homeHandler)
	router.HandleFunc("/gameboard/games", gameboardHandler(gdb)).Methods(http.MethodGet)

	// TODO gérer le port correctement
	server := &http.Server{
		Addr:         ":8080",
		Handler:      router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	go func() {
		slog.Info("Server is running...")
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			slog.Error(fmt.Sprintf("Server crash : %v", err), "cause", err.Error())
			os.Exit(1)
		}
		slog.Info("Server is stopped.")
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	shutdownCtx, shutdownFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownFunc()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error(fmt.Sprintf("Server shutdown with error : %v", err), "cause", err.Error())
		os.Exit(1)
	}
	slog.Info("Server shutdown gracefully.")
}

func homeHandler(_ http.ResponseWriter, _ *http.Request) {
	slog.Info("Home controller accessed.")
}

func gameboardHandler(gdb *gameDatabase) func(http.ResponseWriter, *http.Request) {

	// TODO add creation table : DDL
	// TODO insert values

	return func(w http.ResponseWriter, r *http.Request) {
		slog.Info("Gameboard controller accessed.")
		// TODO récupérer la lsite des jeux de la BDD
		var games = []Game{
			{Name: "Abyss", Description: "Very good game.", JacketPath: "/abyss.png"},
		}
		w.Header().Add("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(games)
		if err != nil {
			slog.Error("Json failed to encode games.", "cause", err)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}
