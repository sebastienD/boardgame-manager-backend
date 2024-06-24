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
	db        *pgxpool.Pool
	connected bool
	// TODO à gerer via mutex
	Ready bool
}

func NewGameDatabase() *gameDatabase {
	return &gameDatabase{}
}

// TODO gérer correctement l'access concurrent aux connexions
func (gdb *gameDatabase) Connect(ctx context.Context, connected context.CancelFunc, connUrl string, wait time.Duration) {
	go func(gdb *gameDatabase) {
		for {
			err := gdb.connect(ctx, connUrl)
			if err == nil {
				// TODO gérer via mutex pour les accès concurrents
				gdb.connected = true
				connected()
				return
			}
			slog.Info("Failed to connect to the database, retrying...", "url", connUrl, "wait", wait)
			time.Sleep(wait)
		}
	}(gdb)
}

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
											title VARCHAR(200) NOT NULL, 
											description VARCHAR(500) NOT NULL, 
											nb_players SMALLSERIAL NOT NULL, 
											jacket_path VARCHAR(200) NOT NULL)`)
	if err != nil {
		return errors.Wrap(err, "create table")
	}

	slog.Info("Table created", "name", "boardgames")
	return nil
}

func (gdb *gameDatabase) FillTables(ctx context.Context, boardgames []Boardgame) error {

	// insert rows
	for _, boardgame := range boardgames {
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
	if gdb.connected {
		gdb.db.Close()
	}
}
