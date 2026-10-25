package widgets

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
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
		return wrap(ctx, "weather", "天気", "",
			`<div class="widget-body"><div class="weather-empty">no location set</div></div>`), nil
	}

	cacheKey := lat + "," + lon

	// ── Try cache first ──
	weatherCacheMu.Lock()
	cached := weatherCache[cacheKey]
	if cached != nil && time.Now().Before(cached.expiresAt) {
		weatherCacheMu.Unlock()
		var data openMeteoResponse
		if err := json.Unmarshal(cached.data, &data); err == nil {
			return renderWeather(ctx, data, city), nil
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
			return renderWeather(ctx, *data, city), nil
		}
		return wrap(ctx, "weather", "天気", "",
			`<div class="widget-body"><div class="weather-empty">fetch failed</div></div>`), nil
	}

	// Swapping the whole widget root (not just this body) is the rule for
	// every widget re-render — see wrap in widget.go. The fragment endpoint
	// returns a complete widget, titlebar included, so targeting ".widget"
	// keeps exactly one titlebar; targeting the body would nest a second one.
	placeholder := `<div class="widget-body" hx-get="/api/widget/weather" hx-trigger="load" hx-target="closest .widget" hx-swap="outerHTML"><div class="weather-empty">…</div></div>`
	return wrap(ctx, "weather", "天気", "", placeholder), nil
}

// fetchWeather retrieves and caches Open-Meteo data. Concurrent fetches
// are throttled to one network attempt per 10s per location: when
// several tabs open within the same hour, only the first fragment
// actually fetches; the others re-serve the cached bytes (fine even a
// little stale) instead of piling requests onto the API.
func fetchWeather(cacheKey, lat, lon string) *openMeteoResponse {
	weatherCacheMu.Lock()
	cached := weatherCache[cacheKey]
	if cached != nil && time.Since(cached.lastAttempt) < 10*time.Second {
		weatherCacheMu.Unlock()
		if cached.data != nil {
			var data openMeteoResponse
			if err := json.Unmarshal(cached.data, &data); err == nil {
				return &data
			}
		}
		return nil
	}
	if weatherCache[cacheKey] == nil {
		weatherCache[cacheKey] = &cachedWeather{}
	}
	weatherCache[cacheKey].lastAttempt = time.Now()
	weatherCacheMu.Unlock()

	url := fmt.Sprintf(
		"https://api.open-meteo.com/v1/forecast?latitude=%s&longitude=%s"+
			"&current=temperature_2m,apparent_temperature,weather_code"+
			"&daily=temperature_2m_max,temperature_2m_min,weather_code"+
			"&timezone=auto",
		lat, lon,
	)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	// Cache the raw response until the next hour
	weatherCacheMu.Lock()
	weatherCache[cacheKey] = &cachedWeather{
		data:        body,
		expiresAt:   nextHour(),
		lastAttempt: time.Now(),
	}
	weatherCacheMu.Unlock()

	var data openMeteoResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil
	}
	return &data
}

// renderWeather builds the weather widget HTML from parsed data.
func renderWeather(ctx RenderContext, data openMeteoResponse, city string) template.HTML {
	unit := "°C"
	if data.CurrentUnits.Temperature == "°F" {
		unit = "°F"
	}

	temp := fmt.Sprintf("%.0f%s", data.Current.Temperature, unit)
	feelsLike := fmt.Sprintf("%.0f%s", data.Current.ApparentTemp, unit)
	condition := wmoWeatherDescription(data.Current.WeatherCode)

	high := ""
	low := ""
	if len(data.Daily.TempMax) > 0 {
		high = fmt.Sprintf("%.0f%s", data.Daily.TempMax[0], unit)
	}
	if len(data.Daily.TempMin) > 0 {
		low = fmt.Sprintf("%.0f%s", data.Daily.TempMin[0], unit)
	}

	locationLine := ""
	if city != "" {
		locationLine = fmt.Sprintf(`<div class="weather-city">%s</div>`, htmlEscape(city))
	}

	hilo := ""
	if high != "" && low != "" {
		hilo = fmt.Sprintf(`<div class="weather-hilo">H %s &middot; L %s</div>`, high, low)
	}

	inner := fmt.Sprintf(`<div class="widget-body"><div class="weather-block">
  <div class="weather-temp">%s</div>
  <div class="weather-condition">%s</div>
  <div class="weather-feels">feels like %s</div>
  %s
  %s
</div></div>`, temp, condition, feelsLike, hilo, locationLine)

	return wrap(ctx, "weather", "天気", "", inner)
}

// wmoWeatherDescription maps WMO weather codes to human-readable strings.
// See https://open-meteo.com/en/docs#weathervariables
func wmoWeatherDescription(code int) string {
	switch {
	case code == 0:
		return "clear sky"
	case code == 1:
		return "mainly clear"
	case code == 2:
		return "partly cloudy"
	case code == 3:
		return "overcast"
	case code >= 45 && code <= 48:
		return "foggy"
	case code >= 51 && code <= 55:
		return "drizzle"
	case code >= 56 && code <= 57:
		return "freezing drizzle"
	case code >= 61 && code <= 65:
		return "rain"
	case code >= 66 && code <= 67:
		return "freezing rain"
	case code >= 71 && code <= 75:
		return "snow"
	case code == 77:
		return "snow grains"
	case code >= 80 && code <= 82:
		return "rain showers"
	case code >= 85 && code <= 86:
		return "snow showers"
	case code >= 95 && code <= 99:
		return "thunderstorm"
	default:
		return "unknown"
	}
}
