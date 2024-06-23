package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"
)

type gameDatabase struct {
	db *pgxpool.Pool
	//conn *pgx.Conn
	// retryer
}

func NewGameDatabase() *gameDatabase {
	return &gameDatabase{}
}

// TODO gérer correctement l'access concurrent aux connexions
func (gdb *gameDatabase) Connect(ctx context.Context, ready context.CancelFunc, connUrl string, wait time.Duration) {
	go func(gdb *gameDatabase) {
		for {
			err := gdb.connect(ctx, connUrl)
			if err == nil {
				ready()
				return
			}
			slog.Info("Failed to connect to the database, retrying...", "url", connUrl, "wait", wait)
			time.Sleep(wait)
		}
	}(gdb)
}

// TODO gérer correctement l'access concurrent aux connexions
func (gdb *gameDatabase) connect(ctx context.Context, connUrl string) error {

	db, err := pgxpool.New(ctx, connUrl)
	if err != nil {
		return errors.Wrap(err, "create connection pool")
	}

	err = db.Ping(ctx)
	if err != nil {
		return errors.Wrap(err, "ping")
	}

	gdb.db = db

	slog.Info("Successfully connected to database", "url", connUrl)

	return nil
}

// TOD gérer l'access concurrent
func (gdb *gameDatabase) CreateTables(ctx context.Context) error {

	// check table
	// TODO ne marche pas :-()
	var exists bool
	if err := gdb.db.QueryRow(ctx, "SELECT EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename = 'boargames' );").Scan(&exists); err != nil {
		return errors.Wrap(err, "select to check existing table")
	}
	if exists {
		slog.Info("Table already exists, nothing to do.")
		return nil
	}

	// create table
	_, err := gdb.db.Exec(ctx, `CREATE TABLE IF NOT EXISTS boardgames (
											id SERIAL PRIMARY KEY, 
											title VARCHAR(50) NOT NULL, 
											description VARCHAR(50) NOT NULL, 
											nb_players SMALLSERIAL NOT NULL, 
											jacket_path VARCHAR(50) NOT NULL)`)
	if err != nil {
		return errors.Wrap(err, "create table")
	}

	slog.Info("Table created", "name", "boardgames")
	return nil
}

func (gdb *gameDatabase) FillTables(ctx context.Context) error {

	// insert rows
	for _, boardgame := range []Boardgame{
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
	} {
		query := `INSERT INTO boardgames (title,description,nb_players,jacket_path) VALUES (@title, @desc, @nbPlayers, @jacketPath)`
		args := pgx.NamedArgs{
			"title":      boardgame.Title,
			"desc":       boardgame.Description,
			"nbPlayers":  boardgame.NbPlayers,
			"jacketPath": boardgame.JacketPath,
		}
		_, err := gdb.db.Exec(ctx, query, args)
		if err != nil {
			return errors.Wrap(err, "insert boardgame row")
		}
	}
	slog.Info("Boardgames inserted")
	return nil
}

func (gdb *gameDatabase) GetBoardgames(ctx context.Context) ([]Boardgame, error) {

	rows, err := gdb.db.Query(ctx, "SELECT id,title,description,nb_players,jacket_path FROM boardgames")
	if err != nil {
		return nil, errors.Wrap(err, "select all boardgames")
	}
	defer rows.Close()

	boardgames := []Boardgame{}
	for rows.Next() {
		var boardgame Boardgame
		err = rows.Scan(&boardgame.Id, &boardgame.Title, &boardgame.Description, &boardgame.NbPlayers, &boardgame.JacketPath)
		if err != nil {
			return nil, errors.Wrap(err, "scan boardgame")
		}

		boardgames = append(boardgames, boardgame)
	}
	return boardgames, nil
}

func (gdb *gameDatabase) Close() {
	gdb.db.Close()
}
