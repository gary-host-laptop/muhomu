package store

// BookmarkLink is one saved bookmark. The JSON tags define the widget
// data-state shape used by client-side CRUD and round-tripped on POST.
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

type QuickItem struct {
	Label   string `json:"label"`
	URL     string `json:"url"`
	Favicon string `json:"favicon"`
}

type RecentItem struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type Quote struct {
	ID     int    `json:"id,omitempty"`
	Text   string `json:"text"`
	Author string `json:"author"`
}
