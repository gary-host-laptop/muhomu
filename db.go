package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

var db *sql.DB

func initDB(path string) error {
	var err error
	db, err = sql.Open("sqlite", path+"?_journal=WAL&_timeout=5000")
	if err != nil {
		return err
	}

	// Schema contains only user-produced content.
	// All configuration lives in config.yaml.
	schema := `
	CREATE TABLE IF NOT EXISTS bookmarks (
		id           INTEGER PRIMARY KEY AUTOINCREMENT,
		folder       TEXT    NOT NULL,
		label        TEXT    NOT NULL,
		url          TEXT    NOT NULL,
		favicon      TEXT    NOT NULL DEFAULT '',
		position     INTEGER NOT NULL DEFAULT 0,
		folder_order INTEGER NOT NULL DEFAULT 0
	);

	CREATE TABLE IF NOT EXISTS quick_access (
		id       INTEGER PRIMARY KEY AUTOINCREMENT,
		label    TEXT    NOT NULL,
		url      TEXT    NOT NULL,
		favicon  TEXT    NOT NULL DEFAULT '',
		position INTEGER NOT NULL DEFAULT 0
	);

	CREATE TABLE IF NOT EXISTS recent (
		id       INTEGER PRIMARY KEY AUTOINCREMENT,
		name     TEXT    NOT NULL,
		url      TEXT    NOT NULL,
		added_at INTEGER NOT NULL DEFAULT (strftime('%s','now'))
	);

	CREATE TABLE IF NOT EXISTS notes (
		id      INTEGER PRIMARY KEY CHECK (id = 1),
		content TEXT    NOT NULL DEFAULT ''
	);

	CREATE TABLE IF NOT EXISTS quotes (
		id     INTEGER PRIMARY KEY AUTOINCREMENT,
		text   TEXT NOT NULL,
		author TEXT NOT NULL DEFAULT ''
	);

	CREATE VIRTUAL TABLE IF NOT EXISTS bookmarks_fts USING fts5(
		label, url, folder, content='bookmarks', content_rowid='id'
	);

	CREATE TRIGGER IF NOT EXISTS bookmarks_ai AFTER INSERT ON bookmarks BEGIN
		INSERT INTO bookmarks_fts(rowid, label, url, folder)
		VALUES (new.id, new.label, new.url, new.folder);
	END;
	CREATE TRIGGER IF NOT EXISTS bookmarks_ad AFTER DELETE ON bookmarks BEGIN
		INSERT INTO bookmarks_fts(bookmarks_fts, rowid, label, url, folder)
		VALUES ('delete', old.id, old.label, old.url, old.folder);
	END;
	CREATE TRIGGER IF NOT EXISTS bookmarks_au AFTER UPDATE ON bookmarks BEGIN
		INSERT INTO bookmarks_fts(bookmarks_fts, rowid, label, url, folder)
		VALUES ('delete', old.id, old.label, old.url, old.folder);
		INSERT INTO bookmarks_fts(rowid, label, url, folder)
		VALUES (new.id, new.label, new.url, new.folder);
	END;

	INSERT OR IGNORE INTO notes (id, content) VALUES (1, '');
	`

	_, err = db.Exec(schema)
	return err
}

// ── NOTES ─────────────────────────────────────────────────────

func getNotes() (string, error) {
	var content string
	err := db.QueryRow("SELECT content FROM notes WHERE id = 1").Scan(&content)
	return content, err
}

func setNotes(content string) error {
	_, err := db.Exec("INSERT OR REPLACE INTO notes (id, content) VALUES (1, ?)", content)
	return err
}

// ── BOOKMARKS ─────────────────────────────────────────────────

type BookmarkLink struct {
	ID    int    `json:"id,omitempty"`
	Label string `json:"label"`
	URL   string `json:"url"`
	Fav   string `json:"fav"`
}

type BookmarkFolder struct {
	Folder string         `json:"folder"`
	Links  []BookmarkLink `json:"links"`
}

func getBookmarks() ([]BookmarkFolder, error) {
	rows, err := db.Query(`
		SELECT id, folder, label, url, favicon, folder_order, position
		FROM bookmarks ORDER BY folder_order, position
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	folderMap := map[string]*BookmarkFolder{}
	folderOrder := []string{}

	for rows.Next() {
		var id int
		var folder, label, url, favicon string
		var folderOrd, pos int
		if err := rows.Scan(&id, &folder, &label, &url, &favicon, &folderOrd, &pos); err != nil {
			continue
		}
		if _, exists := folderMap[folder]; !exists {
			folderMap[folder] = &BookmarkFolder{Folder: folder, Links: []BookmarkLink{}}
			folderOrder = append(folderOrder, folder)
		}
		// pos == -1 is a sentinel row for empty folders
		if pos >= 0 && label != "" {
			folderMap[folder].Links = append(folderMap[folder].Links, BookmarkLink{
				ID: id, Label: label, URL: url, Fav: favicon,
			})
		}
	}

	result := make([]BookmarkFolder, 0, len(folderOrder))
	for _, f := range folderOrder {
		result = append(result, *folderMap[f])
	}
	return result, nil
}

func setBookmarks(folders []BookmarkFolder) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM bookmarks"); err != nil {
		return err
	}

	stmt, err := tx.Prepare(`
		INSERT INTO bookmarks (folder, label, url, favicon, folder_order, position)
		VALUES (?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for fi, folder := range folders {
		if len(folder.Links) == 0 {
			// sentinel row preserves empty folders
			if _, err := stmt.Exec(folder.Folder, "", "", "", fi, -1); err != nil {
				return err
			}
			continue
		}
		for li, link := range folder.Links {
			if _, err := stmt.Exec(folder.Folder, link.Label, link.URL, link.Fav, fi, li); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

func searchBookmarks(q string) ([]BookmarkLink, error) {
	rows, err := db.Query(`
		SELECT b.id, b.folder, b.label, b.url, b.favicon
		FROM bookmarks b
		JOIN bookmarks_fts fts ON b.id = fts.rowid
		WHERE bookmarks_fts MATCH ?
		ORDER BY rank
		LIMIT 20
	`, q+"*")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []BookmarkLink
	for rows.Next() {
		var id int
		var folder, label, url, favicon string
		if err := rows.Scan(&id, &folder, &label, &url, &favicon); err != nil {
			continue
		}
		results = append(results, BookmarkLink{
			ID:    id,
			Label: label,
			URL:   fmt.Sprintf("[%s] %s", folder, url),
			Fav:   favicon,
		})
	}
	return results, nil
}

// ── QUICK ACCESS ──────────────────────────────────────────────

type QuickItem struct {
	Label   string `json:"label"`
	URL     string `json:"url"`
	Favicon string `json:"favicon"`
}

func getQuickAccess() ([]QuickItem, error) {
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

func setQuickAccess(items []QuickItem) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM quick_access"); err != nil {
		return err
	}

	stmt, err := tx.Prepare("INSERT INTO quick_access (label, url, favicon, position) VALUES (?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for i, item := range items {
		if _, err := stmt.Exec(item.Label, item.URL, item.Favicon, i); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ── RECENT ────────────────────────────────────────────────────

type RecentItem struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

func getRecent() ([]RecentItem, error) {
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

func setRecent(items []RecentItem) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM recent"); err != nil {
		return err
	}

	stmt, err := tx.Prepare("INSERT INTO recent (name, url, added_at) VALUES (?, ?, strftime('%s','now'))")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, item := range items {
		if _, err := stmt.Exec(item.Name, item.URL); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ── QUOTES ────────────────────────────────────────────────────

type Quote struct {
	ID     int    `json:"id,omitempty"`
	Text   string `json:"text"`
	Author string `json:"author"`
}

func getQuotes() ([]Quote, error) {
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

func addQuote(text, author string) (Quote, error) {
	res, err := db.Exec("INSERT INTO quotes (text, author) VALUES (?, ?)", text, author)
	if err != nil {
		return Quote{}, err
	}
	id, _ := res.LastInsertId()
	return Quote{ID: int(id), Text: text, Author: author}, nil
}

func deleteQuote(id int) error {
	_, err := db.Exec("DELETE FROM quotes WHERE id = ?", id)
	return err
}

func logFatal(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
