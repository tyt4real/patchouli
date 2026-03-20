package web

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"patchy/database"

	_ "github.com/mattn/go-sqlite3"
)

func StartServer(addr, dbPath string) error {
	db, err := openDB(dbPath)
	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))
	mux.Handle("/attachement/", http.StripPrefix("/attachement/", http.FileServer(http.Dir("data/images"))))
	mux.HandleFunc("/", handleBoards(db))
	mux.HandleFunc("/board/", handleBoardOrThread(db))

	log.Printf("web UI listening on %s", addr)
	return http.ListenAndServe(addr, mux)
}

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

func handleBoardOrThread(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		trimmed := strings.TrimPrefix(r.URL.Path, "/board/")
		parts := strings.SplitN(strings.Trim(trimmed, "/"), "/", 2)

		boardName := parts[0]
		if boardName == "" {
			http.NotFound(w, r)
			return
		}

		if len(parts) == 2 && parts[1] != "" {
			threadNo, err := strconv.Atoi(parts[1])
			if err != nil {
				http.NotFound(w, r)
				return
			}
			handleThread(db, w, r, boardName, threadNo)
			return
		}

		handleBoard(db, w, r, boardName)
	}
}

func handleBoard(db *sql.DB, w http.ResponseWriter, r *http.Request, boardName string) {
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

func handleThread(db *sql.DB, w http.ResponseWriter, r *http.Request, boardName string, threadNo int) {
	threadID, err := database.GetThreadIDByBoardAndNo(db, boardName, threadNo)
	if err != nil {
		http.NotFound(w, r)
		log.Printf("thread not found /%s/%d: %v", boardName, threadNo, err)
		return
	}

	posts, err := database.GetPostsByThread(db, threadID)
	if err != nil {
		http.Error(w, "failed to load posts", http.StatusInternalServerError)
		log.Printf("error getting posts for thread %d: %v", threadID, err)
		return
	}

	postIDs := make([]int, len(posts))
	for i, p := range posts {
		postIDs[i] = p.ID
	}

	images, err := database.GetImagesByPostIDs(db, postIDs)
	if err != nil {
		http.Error(w, "failed to load images", http.StatusInternalServerError)
		log.Printf("error getting images for thread %d: %v", threadID, err)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := threadPage(boardName, threadNo, posts, images).Render(r.Context(), w); err != nil {
		log.Printf("render error (thread /%s/%d): %v", boardName, threadNo, err)
	}
}
