package main

import (
	"database/sql"
	"fmt"
	"log"
	"patchy/config"
	"patchy/crawler"
	"patchy/database"
	"patchy/utils"
	"path/filepath"
	"strings"
)

func main() {
	db, err := database.InitDB()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Loaded configuration for %d boards with cooldown of %d seconds.\n", len(cfg.Boards), cfg.CooldownSeconds)

	for _, boardConfig := range cfg.Boards {
		log.Printf("Archiving board: %s from %s\n", boardConfig.Name, boardConfig.SiteURL)

		boardID, err := database.InsertBoard(db, boardConfig.Name, boardConfig.SiteURL)
		if err != nil {
			log.Printf("Error inserting board %s: %v\n", boardConfig.Name, err)
			continue
		}

		allThreads, err := crawler.GetCatalog(boardConfig, cfg)
		if err != nil {
			log.Printf("Error fetching catalog for board %s: %v\n", boardConfig.Name, err)
			continue
		}

		log.Printf("Found %d threads in catalog for board %s\n", len(allThreads), boardConfig.Name)

		for _, thread := range allThreads {
			log.Printf("Processing thread #%d (OP: %s) on board %s\n", thread.No, thread.Subject, boardConfig.Name)

			threadID, err := database.InsertThread(db, database.Thread{
				BoardID:      boardID,
				ThreadNo:     thread.No,
				LastModified: thread.LastModified,
				OpPostID:     0,
			})
			if err != nil {
				log.Printf("Error inserting thread %d for board %s: %v\n", thread.No, boardConfig.Name, err)
				continue
			}

			threadPosts, err := crawler.GetThreadPostsFromHTML(boardConfig, thread.No, cfg)
			if err != nil {
				log.Printf("Error fetching and parsing HTML for thread #%d: %v\n", thread.No, err)
				continue
			}

			log.Printf("Fetched %d posts from HTML for thread #%d\n", len(threadPosts), thread.No)

			for i, post := range threadPosts {
				postID, err := database.InsertPost(db, database.Post{
					ThreadID:    threadID,
					PostNo:      post.No,
					Resto:       post.Resto,
					Time:        post.Time,
					Name:        post.Name,
					Trip:        post.Trip,
					IDCode:      post.ID,
					Capcode:     post.Capcode,
					Country:     post.Country,
					CountryName: post.CountryName,
					Subject:     post.Subject,
					Comment:     post.Comment,
				})
				if err != nil {
					log.Printf("Error inserting post %d for thread %d: %v\n", post.No, threadID, err)
					continue
				}

				if i == 0 {
					_, err := db.Exec("UPDATE threads SET op_post_id = ? WHERE id = ?", postID, threadID)
					if err != nil {
						log.Printf("Error updating OpPostID for thread %d: %v\n", threadID, err)
					}
				}

				if post.Tim != "" && post.Ext != "" {
					imageLocalPath, _, err := downloadAndSaveImage(db, boardConfig, post, postID)
					if err != nil {
						log.Printf("Error downloading main image for post %d: %v\n", post.No, err)
					} else {
						log.Printf("Downloaded main image for post %d to %s\n", post.No, imageLocalPath)
					}
				}

				for _, extraFile := range post.ExtraFiles {
					if extraFile.Tim != "" && extraFile.Ext != "" {

						extraFilePost := crawler.Post{
							No:          post.No,
							Tim:         extraFile.Tim,
							Filename:    extraFile.Filename,
							Ext:         extraFile.Ext,
							Fsize:       extraFile.Fsize,
							Md5:         extraFile.Md5,
							W:           extraFile.W,
							H:           extraFile.H,
							TnW:         extraFile.TnW,
							TnH:         extraFile.TnH,
							FileDeleted: extraFile.FileDeleted,
							Spoiler:     extraFile.Spoiler,
						}
						imageLocalPath, _, err := downloadAndSaveImage(db, boardConfig, extraFilePost, postID)
						if err != nil {
							log.Printf("Error downloading extra image %s for post %d: %v\n", extraFile.Tim, post.No, err)
						} else {
							log.Printf("Downloaded extra image %s for post %d to %s\n", extraFile.Tim, post.No, imageLocalPath)
						}
					}
				}
			}
		}
		log.Println("Archiving process completed.")
	}
}
func downloadAndSaveImage(db *sql.DB, boardConfig config.BoardConfig, post crawler.Post, postID int) (string, string, error) {
	imageURL := post.ImageURL
	thumbnailURL := post.ThumbnailURL

	imageDir := filepath.Join("data", "images", boardConfig.Name)

	imageBase := filepath.Base(imageURL)
	if imageBase == "" || imageBase == "." || imageBase == string(filepath.Separator) {

		if post.Filename != "" {
			imageBase = post.Filename
		} else if post.Tim != "" && post.Ext != "" {
			imageBase = post.Tim + post.Ext
		} else {

			ext := post.Ext
			if ext == "" {

				if idx := strings.LastIndex(imageURL, "."); idx != -1 && idx < len(imageURL)-1 {
					ext = imageURL[idx:]
				}
			}
			imageBase = fmt.Sprintf("%d_%d%s", post.No, post.Time, ext)
		}
	}

	imageLocalPath := filepath.Join(imageDir, imageBase)

	thumbnailFileName := filepath.Base(thumbnailURL)
	if thumbnailFileName == "" || thumbnailFileName == "." || thumbnailFileName == string(filepath.Separator) {
		if post.Tim != "" && post.Ext != "" {

			thumbnailFileName = post.Tim + "s" + post.Ext
		} else {
			thumbnailFileName = imageBase
		}
	}
	thumbnailLocalPath := filepath.Join(imageDir, thumbnailFileName)

	err := utils.DownloadFile(imageLocalPath, imageURL)
	if err != nil {
		return "", "", fmt.Errorf("failed to download image %s: %w", imageURL, err)
	}

	err = utils.DownloadFile(thumbnailLocalPath, thumbnailURL)
	if err != nil {
		log.Printf("Warning: Failed to download thumbnail %s: %v (might not exist)\n", thumbnailURL, err)
		thumbnailLocalPath = ""
	}

	_, err = database.InsertImage(db, database.Image{
		PostID:             postID,
		Tim:                post.Tim,
		Filename:           post.Filename,
		Ext:                post.Ext,
		Fsize:              post.Fsize,
		Md5:                post.Md5,
		Width:              post.W,
		Height:             post.H,
		ThumbnailWidth:     post.TnW,
		ThumbnailHeight:    post.TnH,
		IsFileDeleted:      post.FileDeleted,
		IsSpoiler:          post.Spoiler,
		LocalPath:          imageLocalPath,
		LocalThumbnailPath: thumbnailLocalPath,
	})
	if err != nil {
		return "", "", fmt.Errorf("failed to insert image metadata for post %d: %w", post.No, err)
	}
	return imageLocalPath, thumbnailLocalPath, nil
}
