package store

import "time"

// Recent returns the eight most recently saved items.
func (s *Store) Recent() ([]RecentItem, error) {
	rows, err := s.db.Query("SELECT name, url FROM recent ORDER BY added_at DESC LIMIT 8")
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

// SetRecent replaces the recent list in one transaction.
func (s *Store) SetRecent(items []RecentItem) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM recent"); err != nil {
		return err
	}

	stmt, err := tx.Prepare("INSERT INTO recent (name, url, added_at) VALUES (?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	// Insert with decreasing timestamps so SELECT ... ORDER BY added_at DESC
	// returns the items in the order the client sent them (the quick-add
	// flow unshifts the newest item to the front). Millisecond precision
	// avoids ties between rapid saves within the same second.
	for i, item := range items {
		if _, err := stmt.Exec(item.Name, item.URL, time.Now().UnixMilli()-int64(i)); err != nil {
			return err
		}
	}
	return tx.Commit()
}
