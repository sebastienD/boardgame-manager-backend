package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"
)

const (
	DEFAULT_DATABASE_URL = "postgres://zenika:secret@localhost:5432/gameboardManagerDB"
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
func (gdb *gameDatabase) Connect(ctx context.Context) error {

	// TODO à revoir
	connUrl := envValue("DATABASE_URL", DEFAULT_DATABASE_URL)
	db, err := pgxpool.New(ctx, connUrl)
	if err != nil {
		return errors.Wrap(err, "create connection pool")
	}

	gdb.db = db

	slog.Info("Successfully connected to database", "url", connUrl)

	return nil
}

func (gdb *gameDatabase) Ping(ctx context.Context) error {
	return gdb.db.Ping(ctx)
}

func envValue(key string, defaultValue string) string {
	// TODO use go get github.com/joho/godotenv ?
	if val, exists := os.LookupEnv(key); exists {
		return val
	}
	slog.Info(fmt.Sprintf("%s env varible isn't defined.", key))
	return defaultValue
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
		{Title: "Duel", Description: "Very good", NbPlayers: 2, JacketPath: "/duel.png"},
		{Title: "Codenames", Description: "Very good as well", NbPlayers: 2, JacketPath: "/codenames.jpg"},
		{Title: "Abyss", Description: "Not so bad", NbPlayers: 4, JacketPath: "/abyss.png"},
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
