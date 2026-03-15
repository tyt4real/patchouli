package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

const (
	databaseFileName = "data/patchy.db"
	schemaSQL        = `
	CREATE TABLE IF NOT EXISTS boards (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		site_url TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS threads (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		board_id INTEGER NOT NULL,
		thread_no INTEGER NOT NULL,
		last_modified INTEGER,
		is_sticky INTEGER DEFAULT 0,
		is_closed INTEGER DEFAULT 0,
		is_archived INTEGER DEFAULT 0,
		op_post_id INTEGER,
		FOREIGN KEY (board_id) REFERENCES boards(id),
		UNIQUE (board_id, thread_no)
	);

	CREATE TABLE IF NOT EXISTS posts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		thread_id INTEGER NOT NULL,
		post_no INTEGER NOT NULL,
		resto INTEGER,
		time INTEGER,
		name TEXT,
		trip TEXT,
		id_code TEXT,
		capcode TEXT,
		country TEXT,
		country_name TEXT,
		subject TEXT,
		comment TEXT,
		FOREIGN KEY (thread_id) REFERENCES threads(id),
		UNIQUE (thread_id, post_no)
	);

	CREATE TABLE IF NOT EXISTS images (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		post_id INTEGER NOT NULL,
		tim TEXT NOT NULL,
		filename TEXT,
		ext TEXT,
		fsize INTEGER,
		md5 TEXT,
		width INTEGER,
		height INTEGER,
		thumbnail_width INTEGER,
		thumbnail_height INTEGER,
		is_file_deleted INTEGER DEFAULT 0,
		is_spoiler INTEGER DEFAULT 0,
		local_path TEXT,
		local_thumbnail_path TEXT,
		FOREIGN KEY (post_id) REFERENCES posts(id),
		UNIQUE (tim, post_id)
	);
	`
)

type Board struct {
	ID      int
	Name    string
	SiteURL string
}

type Thread struct {
	ID           int
	BoardID      int
	ThreadNo     int
	LastModified int
	IsSticky     int
	IsClosed     int
	IsArchived   int
	OpPostID     int
}

type Post struct {
	ID          int
	ThreadID    int
	PostNo      int
	Resto       int
	Time        int
	Name        string
	Trip        string
	IDCode      string
	Capcode     string
	Country     string
	CountryName string
	Subject     string
	Comment     string
}

type Image struct {
	ID                 int
	PostID             int
	Tim                string
	Filename           string
	Ext                string
	Fsize              int
	Md5                string
	Width              int
	Height             int
	ThumbnailWidth     int
	ThumbnailHeight    int
	IsFileDeleted      int
	IsSpoiler          int
	LocalPath          string
	LocalThumbnailPath string
}

func InitDB() (*sql.DB, error) {
	err := os.MkdirAll("data", 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	db, err := sql.Open("sqlite3", databaseFileName)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if _, err = db.Exec(schemaSQL); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	log.Println("Database initialized and schema applied successfully.")
	return db, nil
}

func InsertBoard(db *sql.DB, name, siteURL string) (int, error) {
	var id int
	err := db.QueryRow("SELECT id FROM boards WHERE name = ?", name).Scan(&id)

	if err == sql.ErrNoRows {

		res, insertErr := db.Exec("INSERT INTO boards (name, site_url) VALUES (?, ?)", name, siteURL)
		if insertErr != nil {
			return 0, fmt.Errorf("failed to insert board %s: %w", name, insertErr)
		}
		lastID, _ := res.LastInsertId()
		return int(lastID), nil
	} else if err != nil {
		return 0, fmt.Errorf("failed to query board %s: %w", name, err)
	}

	return id, nil
}

func InsertThread(db *sql.DB, thread Thread) (int, error) {
	log.Printf("Attempting to insert/update thread: %+v\n", thread)
	var existingID int
	var existingLastModified int

	err := db.QueryRow("SELECT id, last_modified FROM threads WHERE board_id = ? AND thread_no = ?",
		thread.BoardID, thread.ThreadNo).Scan(&existingID, &existingLastModified)

	if err == sql.ErrNoRows {
		log.Printf("Thread %d for board %d not found, inserting new thread.\n", thread.ThreadNo, thread.BoardID)
		res, insertErr := db.Exec(
			"INSERT INTO threads (board_id, thread_no, last_modified, is_sticky, is_closed, is_archived, op_post_id) VALUES (?, ?, ?, ?, ?, ?, ?)",
			thread.BoardID, thread.ThreadNo, thread.LastModified, thread.IsSticky, thread.IsClosed, thread.IsArchived, thread.OpPostID,
		)
		if insertErr != nil {
			return 0, fmt.Errorf("failed to insert thread %d for board %d: %w", thread.ThreadNo, thread.BoardID, insertErr)
		}
		lastID, _ := res.LastInsertId()
		log.Printf("Successfully inserted thread %d for board %d with ID %d.\n", thread.ThreadNo, thread.BoardID, lastID)
		return int(lastID), nil
	} else if err != nil {
		return 0, fmt.Errorf("failed to query thread %d for board %d: %w", thread.ThreadNo, thread.BoardID, err)
	}

	if thread.LastModified > existingLastModified {
		log.Printf("Thread %d for board %d exists and has newer content, updating.\n", thread.ThreadNo, thread.BoardID)
		_, updateErr := db.Exec(
			"UPDATE threads SET last_modified = ?, is_sticky = ?, is_closed = ?, is_archived = ?, op_post_id = ? WHERE id = ?",
			thread.LastModified, thread.IsSticky, thread.IsClosed, thread.IsArchived, thread.OpPostID, existingID,
		)
		if updateErr != nil {
			return existingID, fmt.Errorf("failed to update thread %d for board %d: %w", thread.ThreadNo, thread.BoardID, updateErr)
		}
		log.Printf("Successfully updated thread %d for board %d (ID: %d).\n", thread.ThreadNo, thread.BoardID, existingID)
		return existingID, nil
	}

	return existingID, nil
}

func InsertPost(db *sql.DB, post Post) (int, error) {
	var id int
	err := db.QueryRow("SELECT id FROM posts WHERE thread_id = ? AND post_no = ?", post.ThreadID, post.PostNo).Scan(&id)

	if err == sql.ErrNoRows {
		res, insertErr := db.Exec(
			"INSERT INTO posts (thread_id, post_no, resto, time, name, trip, id_code, capcode, country, country_name, subject, comment) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			post.ThreadID, post.PostNo, post.Resto, post.Time, post.Name, post.Trip, post.IDCode, post.Capcode, post.Country, post.CountryName, post.Subject, post.Comment,
		)
		if insertErr != nil {
			return 0, fmt.Errorf("failed to insert post %d for thread %d: %w", post.PostNo, post.ThreadID, insertErr)
		}
		lastID, _ := res.LastInsertId()
		return int(lastID), nil
	} else if err != nil {
		return 0, fmt.Errorf("failed to query post %d for thread %d: %w", post.PostNo, post.ThreadID, err)
	}

	return id, nil
}

func InsertImage(db *sql.DB, image Image) (int, error) {
	var id int
	err := db.QueryRow("SELECT id FROM images WHERE tim = ? AND post_id = ?", image.Tim, image.PostID).Scan(&id)

	if err == sql.ErrNoRows {
		res, insertErr := db.Exec(
			"INSERT INTO images (post_id, tim, filename, ext, fsize, md5, width, height, thumbnail_width, thumbnail_height, is_file_deleted, is_spoiler, local_path, local_thumbnail_path) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			image.PostID, image.Tim, image.Filename, image.Ext, image.Fsize, image.Md5, image.Width, image.Height, image.ThumbnailWidth, image.ThumbnailHeight, image.IsFileDeleted, image.IsSpoiler, image.LocalPath, image.LocalThumbnailPath,
		)
		if insertErr != nil {
			return 0, fmt.Errorf("failed to insert image %s for post %d: %w", image.Tim, image.PostID, insertErr)
		}
		lastID, _ := res.LastInsertId()
		return int(lastID), nil
	} else if err != nil {
		return 0, fmt.Errorf("failed to query image %s for post %d: %w", image.Tim, image.PostID, err)
	}

	_, updateErr := db.Exec(
		"UPDATE images SET local_path = ?, local_thumbnail_path = ? WHERE id = ?",
		image.LocalPath, image.LocalThumbnailPath, id,
	)
	if updateErr != nil {
		return id, fmt.Errorf("failed to update image %s for post %d: %w", image.Tim, image.PostID, updateErr)
	}

	return id, nil
}

// GetBoards returns all boards in the database.
func GetBoards(db *sql.DB) ([]Board, error) {
	rows, err := db.Query("SELECT id, name, site_url FROM boards ORDER BY name ASC")
	if err != nil {
		return nil, fmt.Errorf("failed to query boards: %w", err)
	}
	defer rows.Close()

	var boards []Board
	for rows.Next() {
		var b Board
		if err := rows.Scan(&b.ID, &b.Name, &b.SiteURL); err != nil {
			return nil, fmt.Errorf("failed to scan board row: %w", err)
		}
		boards = append(boards, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	return boards, nil
}

// GetThreadsByBoard returns threads for a given board name.
func GetThreadsByBoard(db *sql.DB, boardName string) ([]Thread, error) {
	query := `SELECT t.id, t.board_id, t.thread_no, t.last_modified, t.is_sticky, t.is_closed, t.is_archived, t.op_post_id
			  FROM threads t
			  JOIN boards b ON t.board_id = b.id
			  WHERE b.name = ?
			  ORDER BY t.last_modified DESC`

	rows, err := db.Query(query, boardName)
	if err != nil {
		return nil, fmt.Errorf("failed to query threads for board %s: %w", boardName, err)
	}
	defer rows.Close()

	var threads []Thread
	for rows.Next() {
		var t Thread
		if err := rows.Scan(&t.ID, &t.BoardID, &t.ThreadNo, &t.LastModified, &t.IsSticky, &t.IsClosed, &t.IsArchived, &t.OpPostID); err != nil {
			return nil, fmt.Errorf("failed to scan thread row: %w", err)
		}
		threads = append(threads, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	return threads, nil
}
