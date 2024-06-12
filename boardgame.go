package main

import "time"

type Boardgame struct {
	Name        string    `json:"title"`
	Description string    `json:"desc"`
	NbPlayers   int       `json:"nb_players"`
	JacketPath  string    `json:"jacket_path"`
	CreatedAt   time.Time `json:"created_at"`
}
