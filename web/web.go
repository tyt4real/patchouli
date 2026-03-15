package web

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"path"

	"patchy/database"

	_ "github.com/mattn/go-sqlite3"
)

// StartServer starts the web UI server.
func StartServer(addr, dbPath string) error {
	db, err := openDB(dbPath)
	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleBoards(db))
	mux.HandleFunc("/board/", handleBoard(db))

	log.Printf("web UI listening on %s", addr)
	return http.ListenAndServe(addr, mux)
}

// openDB opens or initialises the SQLite database.
func openDB(dbPath string) (*sql.DB, error) {
	if dbPath == "" {
		db, err := database.InitDB()
		if err != nil {
			return nil, fmt.Errorf("failed to init db: %w", err)
		}
		return db, nil
	}
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open db %s: %w", dbPath, err)
	}
	return db, nil
}

// handleBoards renders the boards index page.
func handleBoards(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		boards, err := database.GetBoards(db)
		if err != nil {
			http.Error(w, "failed to load boards", http.StatusInternalServerError)
			log.Printf("error getting boards: %v", err)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := boardsPage(boards).Render(r.Context(), w); err != nil {
			log.Printf("render error (boards): %v", err)
		}
	}
}

// handleBoard renders the thread list for a single board.
func handleBoard(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		boardName := path.Base(r.URL.Path)
		if boardName == "" || boardName == "board" {
			http.NotFound(w, r)
			return
		}

		threads, err := database.GetThreadsByBoard(db, boardName)
		if err != nil {
			http.Error(w, "failed to load threads", http.StatusInternalServerError)
			log.Printf("error getting threads for %s: %v", boardName, err)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := boardPage(boardName, threads).Render(r.Context(), w); err != nil {
			log.Printf("render error (board %s): %v", boardName, err)
		}
	}
}
