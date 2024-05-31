package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"

	"github.com/gorilla/mux"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/", homeHandler)
	router.HandleFunc("/gameboard/games", gameboardHandler).Methods(http.MethodGet)
	// TODO gérer le port correctement
	// TODO gérer correctement le interrupt
	slog.Info("Server is running...")
	if err := http.ListenAndServe(":8080", router); err != nil {
		slog.Error("Server crashed.", "cause", err)
		os.Exit(1)
	}
	slog.Info("Server is down.")
}

func homeHandler(_ http.ResponseWriter, _ *http.Request) {
	slog.Info("Home controller accessed.")
}

func gameboardHandler(w http.ResponseWriter, r *http.Request) {
	slog.Info("Gameboard controller accessed.")
	// TODO récupérer l'URL de la BDD d'une var d'env
	// TODO récupérer la lsite des jeux de la BDD
	// TODO récupérer l'URL du path d'une var d'env
	var games = []Game{
		{Name: "Abyss", Description: "Very good game.", JacketPath: "http:localhost:8080/abyss.png"},
	}
	w.Header().Add("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(games)
	if err != nil {
		slog.Error("Json failed to encode games.", "cause", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}
