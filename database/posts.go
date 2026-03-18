package database

import (
	"database/sql"
	"fmt"
	"strings"
)

func GetPostsByThread(db *sql.DB, threadID int) ([]Post, error) {
	const q = `
		SELECT
			id, thread_id, post_no,
			COALESCE(resto, 0),
			COALESCE(time, 0),
			COALESCE(name, ''),
			COALESCE(trip, ''),
			COALESCE(id_code, ''),
			COALESCE(capcode, ''),
			COALESCE(country, ''),
			COALESCE(country_name, ''),
			COALESCE(subject, ''),
			COALESCE(comment, '')
		FROM posts
		WHERE thread_id = ?
		ORDER BY post_no ASC`

	rows, err := db.Query(q, threadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var p Post
		if err := rows.Scan(
			&p.ID, &p.ThreadID, &p.PostNo,
			&p.Resto, &p.Time,
			&p.Name, &p.Trip, &p.IDCode, &p.Capcode,
			&p.Country, &p.CountryName,
			&p.Subject, &p.Comment,
		); err != nil {
			return nil, err
		}
		posts = append(posts, p)
	}
	return posts, rows.Err()
}

func GetImagesByPostIDs(db *sql.DB, postIDs []int) (map[int][]Image, error) {
	result := make(map[int][]Image)
	if len(postIDs) == 0 {
		return result, nil
	}

	placeholders := strings.Repeat("?,", len(postIDs))
	placeholders = placeholders[:len(placeholders)-1]

	q := fmt.Sprintf(`
		SELECT
			id, post_id,
			COALESCE(tim, ''),
			COALESCE(filename, ''),
			COALESCE(ext, ''),
			COALESCE(fsize, 0),
			COALESCE(md5, ''),
			COALESCE(width, 0),
			COALESCE(height, 0),
			COALESCE(thumbnail_width, 0),
			COALESCE(thumbnail_height, 0),
			is_file_deleted,
			is_spoiler,
			COALESCE(local_path, ''),
			COALESCE(local_thumbnail_path, '')
		FROM images
		WHERE post_id IN (%s)
		ORDER BY id ASC`, placeholders)

	args := make([]any, len(postIDs))
	for i, id := range postIDs {
		args[i] = id
	}

	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var img Image
		if err := rows.Scan(
			&img.ID, &img.PostID,
			&img.Tim, &img.Filename, &img.Ext,
			&img.Fsize, &img.Md5,
			&img.Width, &img.Height,
			&img.ThumbnailWidth, &img.ThumbnailHeight,
			&img.IsFileDeleted, &img.IsSpoiler,
			&img.LocalPath, &img.LocalThumbnailPath,
		); err != nil {
			return nil, err
		}
		result[img.PostID] = append(result[img.PostID], img)
	}
	return result, rows.Err()
}

func GetThreadIDByBoardAndNo(db *sql.DB, boardName string, threadNo int) (int, error) {
	var id int
	err := db.QueryRow(`
		SELECT t.id FROM threads t
		JOIN boards b ON b.id = t.board_id
		WHERE b.name = ? AND t.thread_no = ?`,
		boardName, threadNo,
	).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}
