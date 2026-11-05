CREATE TABLE IF NOT EXISTS bookmarks (
	id           INTEGER PRIMARY KEY AUTOINCREMENT,
	folder       TEXT    NOT NULL,
	label        TEXT    NOT NULL,
	url          TEXT    NOT NULL,
	favicon      TEXT    NOT NULL DEFAULT '',
	position     INTEGER NOT NULL DEFAULT 0,
	folder_order INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS quick_access (
	id       INTEGER PRIMARY KEY AUTOINCREMENT,
	label    TEXT    NOT NULL,
	url      TEXT    NOT NULL,
	favicon  TEXT    NOT NULL DEFAULT '',
	position INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS recent (
	id       INTEGER PRIMARY KEY AUTOINCREMENT,
	name     TEXT    NOT NULL,
	url      TEXT    NOT NULL,
	added_at INTEGER NOT NULL DEFAULT (strftime('%s','now'))
);

CREATE TABLE IF NOT EXISTS notes (
	id      INTEGER PRIMARY KEY CHECK (id = 1),
	content TEXT    NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS quotes (
	id     INTEGER PRIMARY KEY AUTOINCREMENT,
	text   TEXT NOT NULL,
	author TEXT NOT NULL DEFAULT ''
);

CREATE VIRTUAL TABLE IF NOT EXISTS bookmarks_fts USING fts5(
	label, url, folder, content='bookmarks', content_rowid='id'
);

CREATE TRIGGER IF NOT EXISTS bookmarks_ai AFTER INSERT ON bookmarks BEGIN
	INSERT INTO bookmarks_fts(rowid, label, url, folder)
	VALUES (new.id, new.label, new.url, new.folder);
END;
CREATE TRIGGER IF NOT EXISTS bookmarks_ad AFTER DELETE ON bookmarks BEGIN
	INSERT INTO bookmarks_fts(bookmarks_fts, rowid, label, url, folder)
	VALUES ('delete', old.id, old.label, old.url, old.folder);
END;
CREATE TRIGGER IF NOT EXISTS bookmarks_au AFTER UPDATE ON bookmarks BEGIN
	INSERT INTO bookmarks_fts(bookmarks_fts, rowid, label, url, folder)
	VALUES ('delete', old.id, old.label, old.url, old.folder);
	INSERT INTO bookmarks_fts(rowid, label, url, folder)
	VALUES (new.id, new.label, new.url, new.folder);
END;

INSERT OR IGNORE INTO notes (id, content) VALUES (1, '');
