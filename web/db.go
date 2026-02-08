package web

import (
	"database/sql"
	"fmt"
	"log"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	Conn *sql.DB
}

type Board struct {
	ID      int
	Name    string
	SiteURL string
}

type Thread struct {
	ID           int
	ThreadNo     int
	LastModified int64
	IsSticky     int
	IsClosed     int
	IsArchived   int
	OpPostID     int
}

type Post struct {
	ID          int
	PostNo      int
	Resto       int
	Time        int64
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

func OpenDB(dbPath string) (*DB, error) {
	if dbPath == "" {
		dbPath = filepath.Join("data", "patchy.db")
	}
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	conn.SetMaxOpenConns(1)

	pragmas := []string{
		"PRAGMA journal_mode = WAL;",
		"PRAGMA synchronous = NORMAL;",
		"PRAGMA foreign_keys = ON;",
	}
	for _, p := range pragmas {
		if _, err := conn.Exec(p); err != nil {
			log.Printf("warning: failed to exec pragma %s: %v", p, err)
		}
	}

	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}

	return &DB{Conn: conn}, nil
}

func (db *DB) Close() error {
	return db.Conn.Close()
}

func (db *DB) GetBoards() ([]Board, error) {
	rows, err := db.Conn.Query("SELECT id, name, site_url FROM boards ORDER BY name ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Board
	for rows.Next() {
		var b Board
		if err := rows.Scan(&b.ID, &b.Name, &b.SiteURL); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, nil
}

func (db *DB) GetBoardByName(name string) (*Board, error) {
	var b Board
	err := db.Conn.QueryRow("SELECT id, name, site_url FROM boards WHERE name = ?", name).Scan(&b.ID, &b.Name, &b.SiteURL)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (db *DB) GetThreads(boardID int, limit, offset int) ([]Thread, error) {
	q := "SELECT id, thread_no, last_modified, is_sticky, is_closed, is_archived, op_post_id FROM threads WHERE board_id = ? ORDER BY last_modified DESC LIMIT ? OFFSET ?"
	rows, err := db.Conn.Query(q, boardID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Thread
	for rows.Next() {
		var t Thread
		if err := rows.Scan(&t.ID, &t.ThreadNo, &t.LastModified, &t.IsSticky, &t.IsClosed, &t.IsArchived, &t.OpPostID); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, nil
}

func (db *DB) GetThreadByNo(boardID, threadNo int) (*Thread, error) {
	var t Thread
	q := "SELECT id, thread_no, last_modified, is_sticky, is_closed, is_archived, op_post_id FROM threads WHERE board_id = ? AND thread_no = ?"
	err := db.Conn.QueryRow(q, boardID, threadNo).Scan(&t.ID, &t.ThreadNo, &t.LastModified, &t.IsSticky, &t.IsClosed, &t.IsArchived, &t.OpPostID)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (db *DB) GetPosts(threadID int) ([]Post, error) {
	q := "SELECT id, post_no, resto, time, name, trip, id_code, capcode, country, country_name, subject, comment FROM posts WHERE thread_id = ? ORDER BY post_no ASC"
	rows, err := db.Conn.Query(q, threadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Post
	for rows.Next() {
		var p Post
		var t sql.NullInt64
		if err := rows.Scan(&p.ID, &p.PostNo, &p.Resto, &t, &p.Name, &p.Trip, &p.IDCode, &p.Capcode, &p.Country, &p.CountryName, &p.Subject, &p.Comment); err != nil {
			return nil, err
		}
		if t.Valid {
			p.Time = t.Int64
		} else {
			p.Time = 0
		}
		out = append(out, p)
	}
	return out, nil
}

func (db *DB) GetImagesForPost(postID int) ([]Image, error) {
	q := "SELECT id, post_id, tim, filename, ext, fsize, md5, width, height, thumbnail_width, thumbnail_height, is_file_deleted, is_spoiler, local_path, local_thumbnail_path FROM images WHERE post_id = ? ORDER BY id ASC"
	rows, err := db.Conn.Query(q, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Image
	for rows.Next() {
		var im Image
		if err := rows.Scan(&im.ID, &im.PostID, &im.Tim, &im.Filename, &im.Ext, &im.Fsize, &im.Md5, &im.Width, &im.Height, &im.ThumbnailWidth, &im.ThumbnailHeight, &im.IsFileDeleted, &im.IsSpoiler, &im.LocalPath, &im.LocalThumbnailPath); err != nil {
			return nil, err
		}
		out = append(out, im)
	}
	return out, nil
}

func (db *DB) GetRepresentativeImage(threadID int) (*Image, error) {
	q := `SELECT i.id, i.post_id, i.tim, i.filename, i.ext, i.fsize, i.md5, i.width, i.height, i.thumbnail_width, i.thumbnail_height, i.is_file_deleted, i.is_spoiler, i.local_path, i.local_thumbnail_path FROM images i JOIN posts p ON i.post_id = p.id WHERE p.thread_id = ? LIMIT 1`
	var im Image
	err := db.Conn.QueryRow(q, threadID).Scan(&im.ID, &im.PostID, &im.Tim, &im.Filename, &im.Ext, &im.Fsize, &im.Md5, &im.Width, &im.Height, &im.ThumbnailWidth, &im.ThumbnailHeight, &im.IsFileDeleted, &im.IsSpoiler, &im.LocalPath, &im.LocalThumbnailPath)
	if err != nil {
		return nil, err
	}
	return &im, nil
}

func FormatTime(unixSec int64) string {
	if unixSec == 0 {
		return ""
	}
	return time.Unix(unixSec, 0).Format(time.RFC3339)
}
