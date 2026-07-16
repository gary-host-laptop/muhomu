package widgets

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"time"
)

type WeatherWidget struct{}

func (w *WeatherWidget) ID() string { return "weather" }

// openMeteoCurrent is the subset of the Open-Meteo API we care about.
type openMeteoResponse struct {
	CurrentUnits struct {
		Temperature string `json:"temperature_2m"`
	} `json:"current_units"`
	Current struct {
		Time          string  `json:"time"`
		Temperature   float64 `json:"temperature_2m"`
		ApparentTemp  float64 `json:"apparent_temperature"`
		WeatherCode   int     `json:"weather_code"`
	} `json:"current"`
	Daily struct {
		TempMax []float64 `json:"temperature_2m_max"`
		TempMin []float64 `json:"temperature_2m_min"`
		WeatherCode []int `json:"weather_code"`
	} `json:"daily"`
}

func (w *WeatherWidget) Render(ctx RenderContext) (template.HTML, error) {
	lat := ctx.LocationLat
	lon := ctx.LocationLon
	city := ctx.LocationCity

	if lat == "" || lon == "" {
		// No location configured — show empty state
		return wrap("weather", "blue", "天気", "",
			`<div class="widget-body"><div class="weather-empty">no location set</div></div>`), nil
	}

	// Fetch from Open-Meteo
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
		return wrap("weather", "blue", "天気", "",
			`<div class="widget-body"><div class="weather-empty">fetch failed</div></div>`), nil
	}
	defer resp.Body.Close()

	var data openMeteoResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return wrap("weather", "blue", "天気", "",
			`<div class="widget-body"><div class="weather-empty">parse failed</div></div>`), nil
	}

	// ── Build display data ──
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

	return wrap("weather", "blue", "天気", "", inner), nil
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
