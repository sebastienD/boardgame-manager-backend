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

var boardgamesDef = []Boardgame{
	{Title: "7 Wonders Duel", Description: "7 wonders, mais version duel", NbPlayers: 2, JacketPath: "https://cdn3.philibertnet.com/343934-large_default/7-wonders-duel.jpg"},
	{Title: "Azul", Description: "Un jeu avec des careaux de mosaïque Portugais", NbPlayers: 4, JacketPath: "https://cdn3.philibertnet.com/402193-large_default/azul.jpg"},
	{Title: "Brass: Lancashire", Description: "Un jeu de stratégie économie a l'ère du rail", NbPlayers: 4, JacketPath: "https://cdn1.philibertnet.com/417603-large_default/brass-lancashire.jpg"},
	{Title: "Carcassonne", Description: "Un jeu a l'ambiance médiévale", NbPlayers: 5, JacketPath: "https://cdn2.philibertnet.com/542823-large_default/carcassonne-vf.jpg"},
	{Title: "Clank!", Description: "Un jeu de deck building", NbPlayers: 4, JacketPath: "https://cdn2.philibertnet.com/361470-large_default/clank.jpg"},
	{Title: "Codenames", Description: "Un jeu d'ambiance et d'espionnage", NbPlayers: 8, JacketPath: "https://cdn1.philibertnet.com/353015-large_default/codenames-vf.jpg"},
	{Title: "Dice Forge", Description: "Un deck building, mais avec des dés", NbPlayers: 4, JacketPath: "https://cdn2.philibertnet.com/369895-large_default/dice-forge.jpg"},
	{Title: "Dixit", Description: "Un jeu de communication", NbPlayers: 4, JacketPath: "https://cdn2.philibertnet.com/509638-large_default/dixit.jpg"},
	{Title: "Hive", Description: "Mieux que les echecs, et avec des insectes", NbPlayers: 2, JacketPath: "https://cdn3.philibertnet.com/476730-large_default/hive-pocket.jpg"},
	{Title: "Jamaica", Description: "Un jeu de course de pirates", NbPlayers: 6, JacketPath: "https://cdn1.philibertnet.com/518554-large_default/jamaica.jpg"},
	{Title: "Skull", Description: "Un jeu de bluff", NbPlayers: 6, JacketPath: "https://cdn3.philibertnet.com/576691-large_default/skull-silver.jpg"},
	{Title: "Timebomb", Description: "Un autre jeu de bluff", NbPlayers: 8, JacketPath: "https://cdn3.philibertnet.com/362353-large_default/time-bomb.jpg"},
}

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
	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		httpPort = "80"
	}

	wait, err := time.ParseDuration(connRetryDuration)
	if err != nil {
		slog.Error("Failed to parse duration", "cause", err.Error())
		os.Exit(1)
	}

	// init database
	gdb := NewGameDatabase()

	// NOTE: This context will be canceled once the DB connection will be ready to use.
	waitConnect, waitConnectFunc := context.WithCancel(context.Background())

	gdb.Connect(context.Background(), waitConnectFunc, connUrl, wait)
	defer gdb.Close()

	go func() {
		<-waitConnect.Done()
		if err := gdb.CreateTables(context.Background()); err != nil {
			slog.Error(fmt.Sprintf("Create tables : %v", err), "cause", err.Error())
			os.Exit(1)
		}

		if err := gdb.FillTables(context.Background(), boardgamesDef); err != nil {
			slog.Error(fmt.Sprintf("Fill tables : %v", err), "cause", err.Error())
			os.Exit(1)
		}
		gdb.Ready = true
	}()

	// define routes and handlers
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/", homeHandler)
	router.HandleFunc("/health", healthHandler).Methods(http.MethodGet)

	s := router.PathPrefix("/boardgames").Subrouter()
	s.Use(mux.CORSMethodMiddleware(s))
	s.Use(readyMiddleware(gdb))
	s.HandleFunc("/", boardgameHandler(gdb)).Methods(http.MethodGet, http.MethodOptions)
	s.HandleFunc("/static", boardgameStaticHandler).Methods(http.MethodGet, http.MethodOptions)

	server := &http.Server{
		Addr:         ":" + httpPort,
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

// TODO encapsuler dans une func, avec passage du contexte + switch ou accès concurrent au ready
func readyMiddleware(gdb *gameDatabase) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if gdb.Ready {
				next.ServeHTTP(w, r)
			} else {
				slog.Error("BDD connection not ready to use.")
				w.WriteHeader(http.StatusTooEarly)
			}
		})
	}
}

func homeHandler(_ http.ResponseWriter, _ *http.Request) {
	slog.Info("Home controller accessed.")
}

func healthHandler(_ http.ResponseWriter, _ *http.Request) {
	slog.Info("Health controller accessed.")
}

func boardgameHandler(gdb *gameDatabase) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		if r.Method == http.MethodOptions {
			return
		}
		slog.Info("Boardgame controller accessed.")
		w.Header().Add("Content-Type", "application/json")
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

func boardgameStaticHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if req.Method == http.MethodOptions {
		return
	}
	slog.Info("Static BoardGame controller accessed.")
	w.Header().Add("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(boardgamesDef)
	if err != nil {
		slog.Error("Json failed to encode games.", "cause", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}
