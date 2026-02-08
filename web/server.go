package web

import (
	"log"
	"net/http"
	"os"
)

func StartServer(addr string, dbPath string) error {
	if err := loadTemplates(); err != nil {
		return err
	}

	db, err := OpenDB(dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/", BoardsHandler(db))
	mux.HandleFunc("/boards/", func(w http.ResponseWriter, r *http.Request) {
		if len(r.URL.Path) >= len("/boards/") && r.URL.Path == "/boards/" {
			BoardsHandler(db)(w, r)
			return
		}
		if len(r.URL.Path) > 0 && (len(r.URL.Path) > 9 && r.URL.Path[len(r.URL.Path)-7:] == "thread/") {
		}
		if containsThread(r.URL.Path) {
			ThreadHandler(db)(w, r)
		} else {
			BoardHandler(db)(w, r)
		}
	})

	fs := http.FileServer(http.Dir("web/static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	imagesFS := http.FileServer(http.Dir("data/images"))
	mux.Handle("/images/", http.StripPrefix("/images/", imagesFS))

	srv := &http.Server{Addr: addr, Handler: mux}
	log.Printf("Web UI listening on %s", addr)
	return srv.ListenAndServe()
}

func containsThread(p string) bool {
	return len(p) > 8 && (p == "/thread" || (len(p) >= 8 && stringContains(p, "/thread/")))
}

func stringContains(s, sub string) bool {
	return len(s) >= len(sub) && (indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func Run(addr string) {
	if err := StartServer(addr, ""); err != nil {
		if err == http.ErrServerClosed {
			log.Printf("server closed")
			os.Exit(0)
		}
		log.Fatalf("server error: %v", err)
	}
}
