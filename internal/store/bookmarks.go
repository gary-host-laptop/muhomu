package store

import "fmt"

// Bookmarks returns the saved folder tree in display order. Empty folders
// are represented by a sentinel row with position -1.
func (s *Store) Bookmarks() ([]BookmarkFolder, error) {
	rows, err := s.db.Query(`
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
		// pos == -1 is a sentinel row for empty folders.
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

// SetBookmarks replaces the whole bookmark tree in one transaction.
func (s *Store) SetBookmarks(folders []BookmarkFolder) error {
	tx, err := s.db.Begin()
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

// SearchBookmarks runs the FTS query and returns matching links. The URL
// field is prefixed with the folder name for display in the search menu.
func (s *Store) SearchBookmarks(q string) ([]BookmarkLink, error) {
	rows, err := s.db.Query(`
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
