package render

type QuickItem struct {
	Label   string `json:"label"`
	URL     string `json:"url"`
	Favicon string `json:"favicon,omitempty"`
	Fav     string `json:"fav,omitempty"`
}

type BookmarkFolder struct {
	Folder string         `json:"folder"`
	Links  []BookmarkLink `json:"links"`
}

type BookmarkLink struct {
	Label string `json:"label"`
	URL   string `json:"url"`
	Fav   string `json:"fav,omitempty"`
}

type RecentItem struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type Image struct {
	ID       int    `json:"id"`
	Type     string `json:"type"`
	Filename string `json:"filename"`
	URL      string `json:"url"`
}

type Quote struct {
	Text   string `json:"text"`
	Author string `json:"author"`
}

type Word struct {
	K string `json:"k"`
	R string `json:"r"`
	M string `json:"m"`
	L string `json:"l"`
}
