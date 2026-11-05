package main

// appCfg holds the resolved runtime directories. Paths are resolved once
// at startup and are not part of config hot-reload.
var appCfg struct {
	faviconDir      string
	profileDir      string
	bgDir           string
	widgetImagesDir string
	widgetThumbsDir string
	themesDir       string
	staticDir       string
}
