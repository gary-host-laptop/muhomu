package main

import "muhomu/internal/store"

// dataStore is the process-wide persistence handle. It is opened once in
// main and passed to widgets through RenderContext.
var dataStore *store.Store
