package main

type Boardgame struct {
	Id          int    `json:"id"`
	Title       string `json:"title"`
	Description string `json:"desc"`
	NbPlayers   int    `json:"nb_players"`
	JacketPath  string `json:"jacket_path"`
}
