package store

// QuickAccess returns the quick links in display order.
func (s *Store) QuickAccess() ([]QuickItem, error) {
	rows, err := s.db.Query("SELECT label, url, favicon FROM quick_access ORDER BY position")
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

// SetQuickAccess replaces the quick links in one transaction.
func (s *Store) SetQuickAccess(items []QuickItem) error {
	tx, err := s.db.Begin()
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
