package crawler

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"

	"patchy/config"
)

type Post struct {
	No            int    `json:"no"`
	Resto         int    `json:"resto"`
	Sticky        int    `json:"sticky"`
	Closed        int    `json:"closed"`
	Archived      int    `json:"archived"`
	Time          int    `json:"time"`
	Name          string `json:"name"`
	Trip          string `json:"trip"`
	ID            string `json:"id"`
	Capcode       string `json:"capcode"`
	Country       string `json:"country"`
	CountryName   string `json:"country_name"`
	Subject       string `json:"sub"`
	Comment       string `json:"com"`
	Tim           string `json:"tim"`
	Filename      string `json:"filename"`
	Ext           string `json:"ext"`
	Fsize         int    `json:"fsize"`
	Md5           string `json:"md5"`
	W             int    `json:"w"`
	H             int    `json:"h"`
	TnW           int    `json:"tn_w"`
	TnH           int    `json:"tn_h"`
	FileDeleted   int    `json:"filedeleted"`
	Spoiler       int    `json:"spoiler"`
	OmittedPosts  int    `json:"omitted_posts"`
	OmittedImages int    `json:"omitted_images"`
	Replies       int    `json:"replies"`
	Images        int    `json:"images"`
	LastModified  int    `json:"last_modified"`
	ImageURL      string
	ThumbnailURL  string
	ExtraFiles    []struct {
		Tim          string `json:"tim"`
		Filename     string `json:"filename"`
		Ext          string `json:"ext"`
		Fsize        int    `json:"fsize"`
		Md5          string `json:"md5"`
		W            int    `json:"w"`
		H            int    `json:"h"`
		TnW          int    `json:"tn_w"`
		TnH          int    `json:"tn_h"`
		FileDeleted  int    `json:"filedeleted"`
		Spoiler      int    `json:"spoiler"`
		ImageURL     string
		ThumbnailURL string
	} `json:"extra_files"`
}

type Page struct {
	Threads []Post `json:"threads"`
}

func FetchJSON(url string, cfg *config.Config) ([]byte, error) {
	log.Printf("Fetching %s", url)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-OK HTTP status for %s: %s", url, resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body from %s: %w", url, err)
	}

	time.Sleep(time.Duration(cfg.CooldownSeconds) * time.Second)

	return body, nil
}

func FetchHTML(url string, cfg *config.Config) (*http.Response, error) {
	log.Printf("Fetching HTML from %s", url)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL %s: %w", url, err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("received non-OK HTTP status for %s: %s", url, resp.Status)
	}

	time.Sleep(time.Duration(cfg.CooldownSeconds) * time.Second)
	return resp, nil
}

func GetCatalog(board config.BoardConfig, cfg *config.Config) ([]Post, error) {
	url := fmt.Sprintf("%s/%s/catalog.json", board.SiteURL, board.Name)
	data, err := FetchJSON(url, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch catalog for board %s: %w", board.Name, err)
	}

	var pages []Page
	if err := json.Unmarshal(data, &pages); err != nil {
		return nil, fmt.Errorf("failed to unmarshal catalog JSON for board %s: %w", board.Name, err)
	}

	var allThreads []Post
	for _, page := range pages {
		allThreads = append(allThreads, page.Threads...)
	}

	return allThreads, nil
}

func GetThreadPostsFromHTML(boardConfig config.BoardConfig, threadNo int, cfg *config.Config) ([]Post, error) {
	url := fmt.Sprintf("%s/%s/thread/%d.html", boardConfig.SiteURL, boardConfig.Name, threadNo)
	log.Printf("Fetching HTML for thread %d from URL: %s", threadNo, url)
	resp, err := FetchHTML(url, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch HTML for thread %d: %w", threadNo, err)
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML for thread %d: %w", threadNo, err)
	}

	var posts []Post
	threadSel := doc.Find(fmt.Sprintf("#thread_%d", threadNo))
	if threadSel.Length() == 0 {
		return nil, fmt.Errorf("thread container #thread_%d not found", threadNo)
	}
	postSelection := threadSel.Find(".post")
	log.Printf("Found %d potential post elements inside thread_%d", postSelection.Length(), threadNo)

	postSelection.Each(func(i int, s *goquery.Selection) {
		var post Post

		if anchorID, ok := s.Find("a.post_anchor").Attr("id"); ok && anchorID != "" {
			fmt.Sscanf(anchorID, "%d", &post.No)
		} else if postID, ok := s.Attr("id"); ok && postID != "" {
			if strings.HasPrefix(postID, "op_") {
				fmt.Sscanf(postID, "op_%d", &post.No)
			} else if strings.HasPrefix(postID, "reply_") {
				fmt.Sscanf(postID, "reply_%d", &post.No)
			} else {
				fmt.Sscanf(postID, "%d", &post.No)
			}
		} else {

			if href, ok := s.Find(".post_no[href]").First().Attr("href"); ok {
				if idx := strings.LastIndex(href, "#"); idx != -1 && idx+1 < len(href) {
					frag := href[idx+1:]
					for len(frag) > 0 && (frag[0] < '0' || frag[0] > '9') {
						frag = frag[1:]
					}
					fmt.Sscanf(frag, "%d", &post.No)
				}
			}
		}

		if i == 0 {
			post.Resto = 0
		} else {
			post.Resto = threadNo
		}

		post.Name = s.Find(".name").Text()

		timeStr, exists := s.Find("time").Attr("datetime")
		if exists {
			t, err := time.Parse(time.RFC3339, timeStr)
			if err == nil {
				post.Time = int(t.Unix())
			}
		}

		bodySel := s.Find(".body")
		bodySel.Find("script, style").Remove()
		commentHTML, _ := bodySel.Html()
		post.Comment = strings.TrimSpace(commentHTML)

		fileContainer := s.Find(".files .file").First()
		if fileContainer.Length() > 0 {
			imgAnchorSelection := fileContainer.Find("a[href*=\"/src/\"]")
			imgAnchorURL, imgAnchorExists := imgAnchorSelection.Attr("href")
			imgSrc, imgSrcExists := fileContainer.Find(".post-image").Attr("src")

			if imgAnchorExists && imgSrcExists {

				if strings.HasPrefix(imgAnchorURL, "/") {
					imgAnchorURL = boardConfig.SiteURL + imgAnchorURL
				}
				if strings.HasPrefix(imgSrc, "/") {
					imgSrc = boardConfig.SiteURL + imgSrc
				}

				post.ImageURL = imgAnchorURL
				post.ThumbnailURL = imgSrc

				base := filepath.Base(imgSrc)
				ext := filepath.Ext(base)
				tim := base[:len(base)-len(ext)]
				if strings.HasSuffix(tim, "s") {
					tim = tim[:len(tim)-1]
				}

				post.Tim = tim
				post.Ext = ext

				filenameLink, filenameLinkExists := fileContainer.Find(".filename-download-link").Attr("download")
				if filenameLinkExists {
					post.Filename = filenameLink
				} else {
					post.Filename = filepath.Base(post.ImageURL)
				}
				fileInfoText := fileContainer.Find(".fileinfo").Text()
				log.Printf("Fileinfo text for post %d: %s", post.No, fileInfoText)
			}
		}
		fileContainer.NextAll().Filter(".file").Each(func(j int, extraFileSelection *goquery.Selection) {
			var extraFile struct {
				Tim          string `json:"tim"`
				Filename     string `json:"filename"`
				Ext          string `json:"ext"`
				Fsize        int    `json:"fsize"`
				Md5          string `json:"md5"`
				W            int    `json:"w"`
				H            int    `json:"h"`
				TnW          int    `json:"tn_w"`
				TnH          int    `json:"tn_h"`
				FileDeleted  int    `json:"filedeleted"`
				Spoiler      int    `json:"spoiler"`
				ImageURL     string
				ThumbnailURL string
			}

			extraFileImgAnchorSelection := extraFileSelection.Find("a[href*=\"/src/\"]")
			extraFileImgAnchorURL, extraFileImgAnchorExists := extraFileImgAnchorSelection.Attr("href")
			extraFileImgSrc, extraFileImgSrcExists := extraFileSelection.Find(".post-image").Attr("src")

			if extraFileImgAnchorExists && extraFileImgSrcExists {
				if strings.HasPrefix(extraFileImgAnchorURL, "/") {
					extraFileImgAnchorURL = boardConfig.SiteURL + extraFileImgAnchorURL
				}
				if strings.HasPrefix(extraFileImgSrc, "/") {
					extraFileImgSrc = boardConfig.SiteURL + extraFileImgSrc
				}

				extraFile.ImageURL = extraFileImgAnchorURL
				extraFile.ThumbnailURL = extraFileImgSrc

				base := filepath.Base(extraFileImgSrc)
				ext := filepath.Ext(base)
				tim := base[:len(base)-len(ext)]
				if strings.HasSuffix(tim, "s") {
					tim = tim[:len(tim)-1]
				}
				extraFile.Tim = tim
				extraFile.Ext = ext

				filenameLink, filenameLinkExists := extraFileSelection.Find(".filename-download-link").Attr("download")
				if filenameLinkExists {
					extraFile.Filename = filenameLink
				} else {
					extraFile.Filename = filepath.Base(extraFile.ImageURL)
				}
			}
			post.ExtraFiles = append(post.ExtraFiles, extraFile)
		})

		if post.No != 0 {
			posts = append(posts, post)
		} else {
			log.Printf("Warning: Skipped post due to missing post number in thread %d", threadNo)
		}
	})

	log.Printf("Successfully parsed %d posts from HTML for thread %d", len(posts), threadNo)

	if len(posts) == 0 {
		return nil, fmt.Errorf("no posts found for thread %d", threadNo)
	}

	return posts, nil
}
