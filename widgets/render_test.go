package widgets

import (
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"muhomu/internal/store"
)

func newWidgetStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.Open(filepath.Join(t.TempDir(), "widgets.db"))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestHeaderWidgetsRenderWithoutChrome(t *testing.T) {
	ctx := RenderContext{
		Username:        `<b>alice</b>`,
		ProfileImageURL: `/api/images/profile/a.png`,
		SearchEngines: []SearchEngine{
			{Name: "ddg", URL: "https://duckduckgo.com/?q=", Default: true},
		},
	}

	clock, err := (&ClockWidget{}).RenderHeader(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(clock), `class="widget"`) {
		t.Fatalf("header clock must not include widget chrome: %s", clock)
	}
	if !strings.Contains(string(clock), `id="clock"`) {
		t.Fatalf("header clock missing clock element: %s", clock)
	}

	identity, err := (&IdentityWidget{}).RenderHeader(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(identity), `<b>alice</b>`) {
		t.Fatalf("username was not escaped: %s", identity)
	}
	if !strings.Contains(string(identity), `&lt;b&gt;alice&lt;/b&gt;`) {
		t.Fatalf("escaped username missing: %s", identity)
	}

	search, err := (&SearchWidget{}).RenderHeader(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(search), `data-url="https://duckduckgo.com/?q="`) {
		t.Fatalf("search engine URL missing: %s", search)
	}
	if !strings.Contains(string(search), `class="engine-btn active"`) {
		t.Fatalf("default engine not marked active: %s", search)
	}

	defaultSearch, err := (&SearchWidget{}).RenderHeader(RenderContext{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(defaultSearch), `data-url="https://duckduckgo.com/?q="`) {
		t.Fatalf("empty engine list did not fall back to duckduckgo: %s", defaultSearch)
	}

	profile, err := (&ProfileWidget{}).RenderHeader(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(profile), `src="/api/images/profile/a.png"`) {
		t.Fatalf("profile image missing: %s", profile)
	}
	if strings.Contains(string(profile), `class="widget"`) {
		t.Fatalf("header profile must not include widget chrome: %s", profile)
	}
}

func TestBookmarksRenderRootAndFolder(t *testing.T) {
	s := newWidgetStore(t)
	err := s.SetBookmarks([]BookmarkFolder{{
		Folder: "work",
		Links: []BookmarkLink{
			{Label: "docs", URL: "https://example.com/docs", Fav: "/api/images/favicons/docs.png"},
			{Label: "remote", URL: "https://remote.example", Fav: "https://icons.example/x.ico"},
		},
	}})
	if err != nil {
		t.Fatal(err)
	}

	w := &BookmarksWidget{}
	root, err := w.Render(RenderContext{
		Store:   s,
		Options: map[string]interface{}{"rows": float64(2)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(root), "work") || !strings.Contains(string(root), "--bm-rows:2") {
		t.Fatalf("root view missing folder or row count: %s", root)
	}
	if !strings.Contains(string(root), "data-state=") {
		t.Fatalf("root view missing serialized state: %s", root)
	}

	folder, err := w.Render(RenderContext{
		Store: s,
		Query: map[string]string{"folder": "0"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(folder), "docs") || !strings.Contains(string(folder), "folder-links") {
		t.Fatalf("folder view missing link or container: %s", folder)
	}
	if !strings.Contains(string(folder), `<img class="fav" src="/api/images/favicons/docs.png" alt="" width="32" height="32">`) {
		t.Fatalf("folder view missing local favicon: %s", folder)
	}
	if !strings.Contains(string(folder), `<span class="fav-letter">R</span>`) {
		t.Fatalf("folder view missing external favicon letter fallback: %s", folder)
	}
	if strings.Contains(string(folder), `<img class="fav" src="https://icons.example`) {
		t.Fatalf("external favicon leaked into tile markup: %s", folder)
	}
	if !strings.Contains(string(folder), "..") {
		t.Fatalf("folder view missing parent tile: %s", folder)
	}
}

func TestNotesRenderEscapesStoredContent(t *testing.T) {
	s := newWidgetStore(t)
	if err := s.SetNotes(`<script>alert("x")</script>`); err != nil {
		t.Fatal(err)
	}

	h, err := (&NotesWidget{}).Render(RenderContext{Store: s})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(h), `<script>`) {
		t.Fatalf("notes content was not escaped: %s", h)
	}
	if !strings.Contains(string(h), `&lt;script&gt;`) {
		t.Fatalf("escaped notes content missing: %s", h)
	}
}

func TestQuoteRenderEscapesTextAndEncodesCurrent(t *testing.T) {
	s := newWidgetStore(t)
	if _, err := s.AddQuote(`<script>alert("x")</script>`, `<b>author</b>`); err != nil {
		t.Fatal(err)
	}

	h, err := (&QuoteWidget{}).Render(RenderContext{
		Store: s,
		RNG:   rand.New(rand.NewSource(1)),
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(h), `<script>`) {
		t.Fatalf("quote text was not escaped: %s", h)
	}
	if !strings.Contains(string(h), `&lt;script&gt;`) {
		t.Fatalf("escaped quote text missing: %s", h)
	}
	if !strings.Contains(string(h), `current=%3Cscript%3Ealert%28%22x%22%29%3C%2Fscript%3E`) {
		t.Fatalf("current quote not URL-encoded: %s", h)
	}
	if !strings.Contains(string(h), `&lt;b&gt;author&lt;/b&gt;`) {
		t.Fatalf("quote author was not escaped: %s", h)
	}
}

func TestQuickAccessRenderUsesLetterForExternalFavicon(t *testing.T) {
	s := newWidgetStore(t)
	err := s.SetQuickAccess([]QuickItem{
		{Label: "remote", URL: "https://remote.example", Favicon: "https://icons.example/favicon.ico"},
		{Label: "local", URL: "https://local.example", Favicon: "/api/images/favicons/local.png"},
	})
	if err != nil {
		t.Fatal(err)
	}

	h, err := (&QuickAccessWidget{}).Render(RenderContext{Store: s})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(h), `<span class="ql-fav-letter">R</span>`) {
		t.Fatalf("external favicon did not fall back to a letter tile: %s", h)
	}
	if !strings.Contains(string(h), `<img class="ql-fav" src="/api/images/favicons/local.png" alt="">`) {
		t.Fatalf("local favicon was not rendered as an image: %s", h)
	}
	if strings.Contains(string(h), `<img class="ql-fav" src="https://icons.example`) {
		t.Fatalf("external favicon URL leaked into the tile markup: %s", h)
	}
}

func TestRecentlyVisitedRenderEscapesItems(t *testing.T) {
	s := newWidgetStore(t)
	err := s.SetRecent([]RecentItem{
		{Name: `<b>later</b>`, URL: `https://example.com/?a=1&b=2`},
	})
	if err != nil {
		t.Fatal(err)
	}

	h, err := (&RecentlyVisitedWidget{}).Render(RenderContext{Store: s})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(h), `<b>later</b>`) {
		t.Fatalf("recent item name was not escaped: %s", h)
	}
	if !strings.Contains(string(h), `&lt;b&gt;later&lt;/b&gt;`) {
		t.Fatalf("escaped recent item name missing: %s", h)
	}
	if !strings.Contains(string(h), `a=1&amp;b=2`) {
		t.Fatalf("recent item URL was not escaped: %s", h)
	}
}

func TestStaticWidgetTemplatesRender(t *testing.T) {
	timer, err := (&TimerWidget{}).Render(RenderContext{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(timer), `id="timer-display"`) || !strings.Contains(string(timer), `id="timer-start"`) {
		t.Fatalf("timer template missing controls: %s", timer)
	}

	rain, err := (&RainWidget{}).Render(RenderContext{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(rain), `id="vol-rain"`) || !strings.Contains(string(rain), `id="vol-thunder"`) {
		t.Fatalf("rain template missing controls: %s", rain)
	}
}

func TestKotobaRender(t *testing.T) {
	h, err := (&KotobaWidget{}).Render(RenderContext{RNG: rand.New(rand.NewSource(1))})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(h), `id="kanji-char"`) || !strings.Contains(string(h), `https://jisho.org/search/`) {
		t.Fatalf("kotoba template missing kanji or link: %s", h)
	}
	if !strings.Contains(string(h), `id="kanji-meaning"`) {
		t.Fatalf("kotoba template missing meaning: %s", h)
	}
}

func TestImageRenderPlaceholderAndNext(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a.png", "b.png"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	h, err := (&ImageWidget{}).Render(RenderContext{
		WidgetImageDir: dir,
		Query:          map[string]string{"current": "a.png"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(h), `src="/api/widget-images/files/b.png"`) {
		t.Fatalf("image template did not advance to b.png: %s", h)
	}
	if !strings.Contains(string(h), `hx-get="/api/widget/image?current=b.png"`) {
		t.Fatalf("image next action missing current filename: %s", h)
	}
	if !strings.Contains(string(h), `<link rel="prefetch" href="/api/widget-images/files/a.png">`) {
		t.Fatalf("image prefetch link missing: %s", h)
	}

	empty, err := (&ImageWidget{}).Render(RenderContext{WidgetImageDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(empty), `id="widget-img-wrap"></div>`) {
		t.Fatalf("empty image template missing placeholder: %s", empty)
	}
	if !strings.Contains(string(empty), `id="widget-img-next" title="next image" disabled`) {
		t.Fatalf("empty image template missing disabled next button: %s", empty)
	}
}

func TestWeatherRender(t *testing.T) {
	var data openMeteoResponse
	data.CurrentUnits.Temperature = "°C"
	data.Current.Temperature = 12.4
	data.Current.ApparentTemp = 10.2
	data.Current.WeatherCode = 0
	data.Daily.TempMax = []float64{15}
	data.Daily.TempMin = []float64{7}

	h, err := renderWeather(RenderContext{}, data, `<b>city</b>`)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`12°C`,
		`clear sky`,
		`feels like 10°C`,
		`H 15°C · L 7°C`,
		`&lt;b&gt;city&lt;/b&gt;`,
	} {
		if !strings.Contains(string(h), want) {
			t.Fatalf("weather output missing %q: %s", want, h)
		}
	}
	if strings.Contains(string(h), `<b>city</b>`) {
		t.Fatalf("weather city was not escaped: %s", h)
	}
}

func TestCalendarRenderUsesRowsAndCurrentDate(t *testing.T) {
	h, err := (&CalendarWidget{}).Render(RenderContext{})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if !strings.Contains(string(h), `id="calendarTable"`) {
		t.Fatalf("calendar table missing: %s", h)
	}
	if got := strings.Count(string(h), "<tr>"); got != 18 {
		t.Fatalf("calendar should render 18 rows, got %d: %s", got, h)
	}
	if !strings.Contains(string(h), strings.ToLower(now.Format("January"))) {
		t.Fatalf("calendar missing current month: %s", h)
	}
	if !strings.Contains(string(h), ">"+strconv.Itoa(now.Year())+"</td>") {
		t.Fatalf("calendar missing current year: %s", h)
	}
}

func TestSplitColumnRender(t *testing.T) {
	s := newWidgetStore(t)
	h, err := (&SplitColumnWidget{}).Render(RenderContext{
		Store: s,
		Options: map[string]interface{}{
			"max-columns": float64(2),
			"widgets": []interface{}{
				map[string]interface{}{"id": "notes"},
				map[string]interface{}{"id": "timer"},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(string(h), `class="split-col"`); got != 2 {
		t.Fatalf("expected 2 split columns, got %d: %s", got, h)
	}
	if !strings.Contains(string(h), `data-widget="notes"`) || !strings.Contains(string(h), `data-widget="timer"`) {
		t.Fatalf("split column missing child widgets: %s", h)
	}
}
