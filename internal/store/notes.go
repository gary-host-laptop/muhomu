package store

// Notes returns the single notes row. An empty database returns "".
func (s *Store) Notes() (string, error) {
	var content string
	err := s.db.QueryRow("SELECT content FROM notes WHERE id = 1").Scan(&content)
	return content, err
}

// SetNotes replaces the single notes row.
func (s *Store) SetNotes(content string) error {
	_, err := s.db.Exec("INSERT OR REPLACE INTO notes (id, content) VALUES (1, ?)", content)
	return err
}
