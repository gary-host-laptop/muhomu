package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"muhomu/internal/media"
	"muhomu/internal/store"
)

func main() {
	port := flag.String("port", "8080", "port to listen on")
	dataDir := flag.String("data", "./data", "path to data directory")
	staticDir := flag.String("static", "./static", "path to static files")
	configPath := flag.String("config", "./data/config.yaml", "path to config file")
	flag.Parse()

	startupCfg := loadConfig(*configPath)
	setConfig(startupCfg)
	quotesPath := filepath.Join(*dataDir, "quotes.yaml")
	loadQuotes(quotesPath)

	// The settings API writes these files — keep the resolved paths
	// around at package level.
	configFilePath = *configPath
	quotesFilePath = quotesPath

	resolve := func(cfgVal, def string) string {
		if cfgVal != "" {
			return cfgVal
		}
		return filepath.Join(*dataDir, def)
	}

	// Data-directory resolution happens once at startup: these paths are
	// also overridable by flags, so they are not part of hot-reload.
	appCfg.profileDir = resolve(startupCfg.ProfileImagesDir, "images/profile")
	appCfg.bgDir = resolve(startupCfg.BGImagesDir, "images/bg")
	appCfg.widgetImagesDir = resolve(startupCfg.WidgetImagesDir, "images/widget")
	appCfg.widgetThumbsDir = filepath.Join(appCfg.widgetImagesDir, "thumbs")
	appCfg.faviconDir = filepath.Join(*dataDir, "images", "favicons")
	appCfg.themesDir = resolve(startupCfg.ThemesDir, "themes")
	appCfg.staticDir = *staticDir

	for _, d := range []string{
		appCfg.profileDir, appCfg.bgDir,
		appCfg.widgetImagesDir, appCfg.faviconDir,
		appCfg.themesDir, appCfg.widgetThumbsDir,
	} {
		os.MkdirAll(d, 0755)
	}

	dbPath := filepath.Join(*dataDir, "muhomu.db")
	st, err := store.Open(dbPath)
	if err != nil {
		log.Fatal("failed to init db:", err)
	}
	dataStore = st
	defer dataStore.Close()

	go migrateFavicons(appCfg.faviconDir)
	go media.EnsureThumbnails(appCfg.widgetImagesDir)
	startStatsCollector()
	go startRSSWarmer()

	// Hot-reload config.yaml: edits to the running config are picked up
	// within a couple of seconds, no restart needed.
	go watchConfig(*configPath)

	if err := parseTemplates(); err != nil {
		log.Fatal(err)
	}

	mux := newMux()

	log.Printf("muhomu running on http://localhost:%s", *port)
	if err := http.ListenAndServe(":"+*port, mux); err != nil {
		log.Fatal(err)
	}
}
