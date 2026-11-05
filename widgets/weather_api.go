package widgets

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"time"
)

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
func renderWeather(ctx RenderContext, data openMeteoResponse, city string) (template.HTML, error) {
	unit := "°C"
	if data.CurrentUnits.Temperature == "°F" {
		unit = "°F"
	}

	view := weatherView{
		Temp:      fmt.Sprintf("%.0f%s", data.Current.Temperature, unit),
		FeelsLike: fmt.Sprintf("%.0f%s", data.Current.ApparentTemp, unit),
		Condition: wmoWeatherDescription(data.Current.WeatherCode),
		City:      city,
	}
	if len(data.Daily.TempMax) > 0 {
		view.High = fmt.Sprintf("%.0f%s", data.Daily.TempMax[0], unit)
	}
	if len(data.Daily.TempMin) > 0 {
		view.Low = fmt.Sprintf("%.0f%s", data.Daily.TempMin[0], unit)
	}

	inner, err := renderWidgetTemplate("weather.tmpl", view)
	if err != nil {
		return "", err
	}
	return wrap(ctx, "weather", "天気", "", string(inner)), nil
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
