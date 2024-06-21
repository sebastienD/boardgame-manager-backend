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

	"github.com/joho/godotenv"

	"github.com/gorilla/mux"
)

func main() {
	// init logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// config
	_ = godotenv.Load()

	connUrl := os.Getenv("DATABASE_URL")
	connRetryDuration := os.Getenv("RETRY_CONNECTION_AFTER_FAILED")
	if connRetryDuration == "" {
		connRetryDuration = "10s"
	}

	wait, err := time.ParseDuration(connRetryDuration)
	if err != nil {
		slog.Error("Failed to parse duration", "cause", err.Error())
		os.Exit(1)
	}

	// init database
	gdb := NewGameDatabase()

	// NOTE: This context will be canceled once the DB connection will be ready to use.
	waitPlease, waitPleaseFunc := context.WithCancel(context.Background())

	gdb.Connect(context.Background(), waitPleaseFunc, connUrl, wait)
	defer gdb.Close()

	go func() {
		<-waitPlease.Done()
		if err := gdb.CreateTables(context.Background()); err != nil {
			slog.Error(fmt.Sprintf("Create tables : %v", err), "cause", err.Error())
			os.Exit(1)
		}

		if err := gdb.FillTables(context.Background()); err != nil {
			slog.Error(fmt.Sprintf("Fill tables : %v", err), "cause", err.Error())
			os.Exit(1)
		}
	}()

	// define routes and handlers
	router := mux.NewRouter().StrictSlash(true)
	router.Use(readyMiddleware)
	router.HandleFunc("/", homeHandler)
	router.HandleFunc("/gameboards", gameboardHandler(gdb)).Methods(http.MethodGet)

	/*
	   // IMPORTANT: you must specify an OPTIONS method matcher for the middleware to set CORS headers
	   r.HandleFunc("/foo", fooHandler).Methods(http.MethodGet, http.MethodPut, http.MethodPatch, http.MethodOptions)
	   r.Use(mux.CORSMethodMiddleware(r))
	*/

	// TODO gérer le port correctement en var d'env
	server := &http.Server{
		Addr:         ":80",
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

// TODO encapsuler dans une func, avec passage du contexte + switch
func readyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//switch ctx.Done()
		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(w, r)
	})
}

func homeHandler(_ http.ResponseWriter, _ *http.Request) {
	slog.Info("Home controller accessed.")
}

func gameboardHandler(gdb *gameDatabase) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		slog.Info("Gameboard controller accessed.")
		w.Header().Add("Content-Type", "application/json")
		// TODO récupérer la lsite des jeux de la BDD
		boardgames, err := gdb.GetBoardgames(r.Context())
		// TODO gestion err.NoRows
		if err != nil {
			slog.Error("Json failed to encode games.", "cause", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		err = json.NewEncoder(w).Encode(boardgames)
		if err != nil {
			slog.Error("Json failed to encode games.", "cause", err)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}
