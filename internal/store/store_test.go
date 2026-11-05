package store

import (
	"path/filepath"
	"testing"
)

func openTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestBookmarkRoundTrip(t *testing.T) {
	s := openTestStore(t)

	in := []BookmarkFolder{
		{Folder: "work", Links: []BookmarkLink{
			{Label: "docs", URL: "https://example.com/docs", Fav: "/api/images/favicons/a.png"},
		}},
		{Folder: "empty", Links: []BookmarkLink{}},
	}
	if err := s.SetBookmarks(in); err != nil {
		t.Fatalf("SetBookmarks: %v", err)
	}
	out, err := s.Bookmarks()
	if err != nil {
		t.Fatalf("Bookmarks: %v", err)
	}
	if len(out) != 2 || out[0].Folder != "work" || out[1].Folder != "empty" {
		t.Fatalf("folders = %#v", out)
	}
	if len(out[0].Links) != 1 || out[0].Links[0].Label != "docs" {
		t.Fatalf("links = %#v", out[0].Links)
	}
	if len(out[1].Links) != 0 {
		t.Fatalf("empty folder links = %#v", out[1].Links)
	}

	results, err := s.SearchBookmarks("docs")
	if err != nil {
		t.Fatalf("SearchBookmarks: %v", err)
	}
	if len(results) != 1 || results[0].Label != "docs" {
		t.Fatalf("search results = %#v", results)
	}
}

func TestQuickAccessRoundTrip(t *testing.T) {
	s := openTestStore(t)
	in := []QuickItem{{Label: "a", URL: "https://a.example", Favicon: "a.png"}}
	if err := s.SetQuickAccess(in); err != nil {
		t.Fatalf("SetQuickAccess: %v", err)
	}
	out, err := s.QuickAccess()
	if err != nil {
		t.Fatalf("QuickAccess: %v", err)
	}
	if len(out) != 1 || out[0].Label != "a" || out[0].URL != "https://a.example" {
		t.Fatalf("items = %#v", out)
	}
}

func TestRecentRoundTripOrder(t *testing.T) {
	s := openTestStore(t)
	in := []RecentItem{
		{Name: "new", URL: "https://new.example"},
		{Name: "old", URL: "https://old.example"},
	}
	if err := s.SetRecent(in); err != nil {
		t.Fatalf("SetRecent: %v", err)
	}
	out, err := s.Recent()
	if err != nil {
		t.Fatalf("Recent: %v", err)
	}
	if len(out) != 2 || out[0].Name != "new" || out[1].Name != "old" {
		t.Fatalf("items = %#v", out)
	}
}

func TestNotesAndQuotes(t *testing.T) {
	s := openTestStore(t)

	if err := s.SetNotes("hello"); err != nil {
		t.Fatalf("SetNotes: %v", err)
	}
	notes, err := s.Notes()
	if err != nil || notes != "hello" {
		t.Fatalf("Notes = %q, %v", notes, err)
	}

	q, err := s.AddQuote("line", "author")
	if err != nil {
		t.Fatalf("AddQuote: %v", err)
	}
	if q.ID == 0 {
		t.Fatal("AddQuote returned zero id")
	}
	quotes, err := s.Quotes()
	if err != nil || len(quotes) != 1 || quotes[0].Text != "line" {
		t.Fatalf("Quotes = %#v, %v", quotes, err)
	}
	if err := s.DeleteQuote(q.ID); err != nil {
		t.Fatalf("DeleteQuote: %v", err)
	}
	quotes, err = s.Quotes()
	if err != nil || len(quotes) != 0 {
		t.Fatalf("Quotes after delete = %#v, %v", quotes, err)
	}
}
