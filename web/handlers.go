package web

import (
	"html/template"
	"log"
	"net/http"
	"path"
	"strconv"
	"strings"
)

var templates *template.Template

func loadTemplates() error {
	tmpl := template.New("templates").Funcs(template.FuncMap{
		"basename": path.Base,
	})
	var err error
	templates, err = tmpl.ParseGlob("web/templates/*.html")
	return err
}
func BoardsHandler(db *DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		boards, err := db.GetBoards()
		if err != nil {
			http.Error(w, "failed to load boards", http.StatusInternalServerError)
			log.Printf("BoardsHandler: %v", err)
			return
		}
		if err := templates.ExecuteTemplate(w, "index", boards); err != nil {
			log.Printf("template error: %v", err)
		}
	}
}

func BoardHandler(db *DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.Path, "/boards/")
		if p == "" {
			http.NotFound(w, r)
			return
		}
		p = strings.TrimSuffix(p, "/")
		boardName := path.Clean(p)
		board, err := db.GetBoardByName(boardName)
		if err != nil {
			http.Error(w, "board not found", http.StatusNotFound)
			return
		}
		threads, err := db.GetThreads(board.ID, 50, 0)
		if err != nil {
			http.Error(w, "failed to load threads", http.StatusInternalServerError)
			log.Printf("GetThreads: %v", err)
			return
		}
		type threadView struct {
			Thread Thread
			Thumb  *Image
		}
		var tvs []threadView
		for _, t := range threads {
			img, _ := db.GetRepresentativeImage(t.ID)
			tvs = append(tvs, threadView{Thread: t, Thumb: img})
		}

		data := struct {
			Board   *Board
			Threads []threadView
		}{
			Board:   board,
			Threads: tvs,
		}

		if err := templates.ExecuteTemplate(w, "board", data); err != nil {
			log.Printf("template board error: %v", err)
		}
	}
}

func ThreadHandler(db *DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) < 3 || parts[0] != "boards" || parts[2] != "thread" {
			http.NotFound(w, r)
			return
		}
		boardName := parts[1]
		threadNoStr := parts[3]
		threadNo, err := strconv.Atoi(threadNoStr)
		if err != nil {
			http.Error(w, "invalid thread number", http.StatusBadRequest)
			return
		}
		board, err := db.GetBoardByName(boardName)
		if err != nil {
			http.Error(w, "board not found", http.StatusNotFound)
			return
		}
		thread, err := db.GetThreadByNo(board.ID, threadNo)
		if err != nil {
			http.Error(w, "thread not found", http.StatusNotFound)
			return
		}
		posts, err := db.GetPosts(thread.ID)
		if err != nil {
			http.Error(w, "failed to load posts", http.StatusInternalServerError)
			return
		}

		type postView struct {
			Post   Post
			Images []Image
		}
		var pvs []postView
		for _, p := range posts {
			imgs, _ := db.GetImagesForPost(p.ID)
			pvs = append(pvs, postView{Post: p, Images: imgs})
		}
		data := struct {
			Board  *Board
			Thread *Thread
			Posts  []postView
		}{
			Board:  board,
			Thread: thread,
			Posts:  pvs,
		}
		if err := templates.ExecuteTemplate(w, "thread", data); err != nil {
			log.Printf("template thread error: %v", err)
		}
	}
}
