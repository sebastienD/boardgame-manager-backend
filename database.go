package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"
)

const (
	DEFAULT_DATABASE_URL = "postgres://zenika:secret@localhost:5432/gameboardManagerDB"
)

type gameDatabase struct {
	conn *pgx.Conn
	// retryer
}

func NewGameDatabase() *gameDatabase {
	return &gameDatabase{}
}

func (gdb *gameDatabase) Connect() error {
	ctx := context.Background()
	// TODO à revoir
	connUrl := envValue("DATABASE_URL", DEFAULT_DATABASE_URL)
	conn, err := pgx.Connect(ctx, connUrl)
	if err != nil {
		return errors.Wrap(err, "open connection")
	}
	defer conn.Close(ctx)
	err = conn.Ping(ctx)
	if err != nil {
		return errors.Wrap(err, "ping database")
	}

	slog.Info(fmt.Sprintf("Successfully connected to database with %s", connUrl))
	gdb.conn = conn

	return nil
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
func (gdb *gameDatabase) CreateAndFillTables(ctx context.Context) error {
	// check table
	var exists bool
	if err := gdb.conn.QueryRow(ctx, "SELECT EXISTS (SELECT FROM pg_tables WHERE  schemaname = 'public' AND tablename = 'boargames' )").Scan(&exists); err != nil {
		return errors.Wrap(err, "select to check existing table")
	}
	if exists {
		slog.Info("Table already exists, nothing to do.")
		return nil
	}

	// create table
	results, err := gdb.conn.Query(ctx, "CREATE TABLE boardgames (id SERIAL PRIMARY KEY, title VARCHAR(100) NOT NULL, title VARCHAR(50) NOT NULL, nb_players SMALLSERIAL NOT NULL, created_at TIMESTAMP NOT NULL)")
	if err != nil {
		return errors.Wrap(err, "create table")
	}
	slog.Info("Table created", "name", "boardgames")

	// insert rows
	for _, boardgame := range []struct {
		title     string
		nbPlayers int
		created   time.Time
	}{
		{"Duel", 2, time.Date(2000, 10, 2, 0, 0, 0, 0, time.Local)},
		{"Codenames", 2, time.Date(2002, 8, 1, 0, 0, 0, 0, time.Local)},
		{"Abyss", 4, time.Date(1990, 5, 9, 0, 0, 0, 0, time.Local)},
	} {
		queryStmt := `INSERT INTO boardgames (title,nb_players,created_at) VALUES ($1, $2, $3) RETURNING $4`

		err := gdb.conn.QueryRow(ctx, queryStmt, &article.Id, &article.Title, &article.Desc, &article.Content).Scan(&article.Id)
		if err != nil {
			log.Println("failed to execute query", err)
			return
		}
	}
	fmt.Println("Mock Articles included in Table", results)
	return nil
}

func (h handler) GetAllArticles(w http.ResponseWriter, r *http.Request) {

	results, err := h.DB.Query("SELECT * FROM articles;")
	if err != nil {
		log.Println("failed to execute query", err)
		w.WriteHeader(500)
		return
	}

	var articles = make([]models.Article, 0)
	for results.Next() {
		var article models.Article
		err = results.Scan(&article.Id, &article.Title, &article.Desc, &article.Content)
		if err != nil {
			log.Println("failed to scan", err)
			w.WriteHeader(500)
			return
		}

		articles = append(articles, article)
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(articles)
}

func main() {
	// urlExample := "postgres://username:password@localhost:5432/database_name"
	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(context.Background())

	var name string
	var weight int64
	err = conn.QueryRow(context.Background(), "select name, weight from widgets where id=$1", 42).Scan(&name, &weight)
	if err != nil {
		fmt.Fprintf(os.Stderr, "QueryRow failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(name, weight)
}

func (gdb *gameDatabase) Close() {
	if gdb.conn != nil {
		if err := gdb.conn.Close(context.Background()); err != nil {
			slog.Error("Cloase databse connection.", "cause", err)
		}
	}
}
