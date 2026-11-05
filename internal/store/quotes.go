package store

// Quotes returns all database-backed quotes in id order.
func (s *Store) Quotes() ([]Quote, error) {
	rows, err := s.db.Query("SELECT id, text, author FROM quotes ORDER BY id")
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

func (s *Store) AddQuote(text, author string) (Quote, error) {
	res, err := s.db.Exec("INSERT INTO quotes (text, author) VALUES (?, ?)", text, author)
	if err != nil {
		return Quote{}, err
	}
	id, _ := res.LastInsertId()
	return Quote{ID: int(id), Text: text, Author: author}, nil
}

func (s *Store) DeleteQuote(id int) error {
	_, err := s.db.Exec("DELETE FROM quotes WHERE id = ?", id)
	return err
}
