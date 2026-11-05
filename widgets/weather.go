package widgets

import (
	"encoding/json"
	"html/template"
	"sync"
	"time"
)

// ── Weather cache (on-the-hour) ───────────────────────────────
// Glance caches weather until the start of the next hour.
// Forecast data doesn't change minute-to-minute, so one fetch per
// hour is more than enough and avoids rate limits.
type cachedWeather struct {
	data        []byte
	expiresAt   time.Time
	lastAttempt time.Time
}

var (
	weatherCache   = map[string]*cachedWeather{}
	weatherCacheMu sync.Mutex
)

// nextHour returns the start of the next calendar hour.
func nextHour() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), now.Hour()+1, 0, 0, 0, now.Location())
}

type WeatherWidget struct{}

func (w *WeatherWidget) ID() string { return "weather" }

type weatherView struct {
	Temp      string
	Condition string
	FeelsLike string
	High      string
	Low       string
	City      string
}

// openMeteoResponse is the subset of the Open-Meteo API we care about.
type openMeteoResponse struct {
	CurrentUnits struct {
		Temperature string `json:"temperature_2m"`
	} `json:"current_units"`
	Current struct {
		Time         string  `json:"time"`
		Temperature  float64 `json:"temperature_2m"`
		ApparentTemp float64 `json:"apparent_temperature"`
		WeatherCode  int     `json:"weather_code"`
	} `json:"current"`
	Daily struct {
		TempMax     []float64 `json:"temperature_2m_max"`
		TempMin     []float64 `json:"temperature_2m_min"`
		WeatherCode []int     `json:"weather_code"`
	} `json:"daily"`
}

func (w *WeatherWidget) Render(ctx RenderContext) (template.HTML, error) {
	lat := ctx.LocationLat
	lon := ctx.LocationLon
	city := ctx.LocationCity

	if lat == "" || lon == "" {
		inner, err := renderWidgetTemplate("weather-empty.tmpl", "no location set")
		if err != nil {
			return "", err
		}
		return wrap(ctx, "weather", "天気", "", string(inner)), nil
	}

	cacheKey := lat + "," + lon

	// ── Try cache first ──
	weatherCacheMu.Lock()
	cached := weatherCache[cacheKey]
	if cached != nil && time.Now().Before(cached.expiresAt) {
		weatherCacheMu.Unlock()
		var data openMeteoResponse
		if err := json.Unmarshal(cached.data, &data); err == nil {
			return renderWeather(ctx, data, city)
		}
		// corrupted cache — fall through to fetch
	} else {
		weatherCacheMu.Unlock()
	}

	// ── Cold or expired. A full-page render must never block on the
	// network: it renders a placeholder that htmx swaps out by fetching
	// the widget fragment — the only path that actually (re)fetches.
	// Previously a new tab with an expired hourly cache froze the page
	// for up to five seconds waiting on Open-Meteo.
	if ctx.Fragment {
		if data := fetchWeather(cacheKey, lat, lon); data != nil {
			return renderWeather(ctx, *data, city)
		}
		inner, err := renderWidgetTemplate("weather-empty.tmpl", "fetch failed")
		if err != nil {
			return "", err
		}
		return wrap(ctx, "weather", "天気", "", string(inner)), nil
	}

	// Swapping the whole widget root (not just this body) is the rule for
	// every widget re-render — see wrap in widget.go. The fragment endpoint
	// returns a complete widget, titlebar included, so targeting ".widget"
	// keeps exactly one titlebar; targeting the body would nest a second one.
	placeholder, err := renderWidgetTemplate("weather-placeholder.tmpl", nil)
	if err != nil {
		return "", err
	}
	return wrap(ctx, "weather", "天気", "", string(placeholder)), nil
}
