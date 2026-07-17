package widgets

import (
	"database/sql"
)

// These types and db functions are the widget package's own interface
// to user-produced data. They mirror the main package's db functions
// but are owned by the widgets package so widgets are self-contained.

type BookmarkLink struct {
	Label string
	URL   string
	Fav   string
}

type BookmarkFolder struct {
	Folder string
	Links  []BookmarkLink
}

type QuickItem struct {
	Label   string
	URL     string
	Favicon string
}

type RecentItem struct {
	Name string
	URL  string
}

type Quote struct {
	ID     int
	Text   string
	Author string
}

type Word struct {
	K string `json:"k"`
	R string `json:"r"`
	M string `json:"m"`
	L string `json:"l"`
}

func dbBookmarks(db *sql.DB) ([]BookmarkFolder, error) {
	rows, err := db.Query(`
		SELECT folder, label, url, favicon, folder_order, position
		FROM bookmarks ORDER BY folder_order, position
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	folderMap := map[string]*BookmarkFolder{}
	folderOrder := []string{}

	for rows.Next() {
		var folder, label, url, favicon string
		var folderOrd, pos int
		if err := rows.Scan(&folder, &label, &url, &favicon, &folderOrd, &pos); err != nil {
			continue
		}
		if _, exists := folderMap[folder]; !exists {
			folderMap[folder] = &BookmarkFolder{Folder: folder}
			folderOrder = append(folderOrder, folder)
		}
		if pos >= 0 && label != "" {
			folderMap[folder].Links = append(folderMap[folder].Links, BookmarkLink{
				Label: label, URL: url, Fav: favicon,
			})
		}
	}

	result := make([]BookmarkFolder, 0, len(folderOrder))
	for _, f := range folderOrder {
		result = append(result, *folderMap[f])
	}
	return result, nil
}

func dbQuickAccess(db *sql.DB) ([]QuickItem, error) {
	rows, err := db.Query("SELECT label, url, favicon FROM quick_access ORDER BY position")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []QuickItem
	for rows.Next() {
		var item QuickItem
		if err := rows.Scan(&item.Label, &item.URL, &item.Favicon); err != nil {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

func dbRecent(db *sql.DB) ([]RecentItem, error) {
	rows, err := db.Query("SELECT name, url FROM recent ORDER BY added_at DESC LIMIT 8")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []RecentItem
	for rows.Next() {
		var item RecentItem
		if err := rows.Scan(&item.Name, &item.URL); err != nil {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

func dbNotes(db *sql.DB) (string, error) {
	var content string
	err := db.QueryRow("SELECT content FROM notes WHERE id = 1").Scan(&content)
	return content, err
}

func dbQuotes(db *sql.DB) ([]Quote, error) {
	rows, err := db.Query("SELECT id, text, author FROM quotes ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var quotes []Quote
	for rows.Next() {
		var q Quote
		if err := rows.Scan(&q.ID, &q.Text, &q.Author); err != nil {
			continue
		}
		quotes = append(quotes, q)
	}
	return quotes, nil
}

// htmlEscape escapes html special chars for safe embedding.
func htmlEscape(s string) string {
	result := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '&':
			result = append(result, '&', 'a', 'm', 'p', ';')
		case '<':
			result = append(result, '&', 'l', 't', ';')
		case '>':
			result = append(result, '&', 'g', 't', ';')
		case '"':
			result = append(result, '&', 'q', 'u', 'o', 't', ';')
		case '\'':
			result = append(result, '&', '#', '3', '9', ';')
		default:
			result = append(result, s[i])
		}
	}
	return string(result)
}
