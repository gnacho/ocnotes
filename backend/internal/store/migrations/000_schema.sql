-- Schema version 0: initial table creation.
CREATE TABLE IF NOT EXISTS notes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL DEFAULT 'New note',
    content TEXT NOT NULL DEFAULT '',
    category TEXT NOT NULL DEFAULT '',
    favorite INTEGER NOT NULL DEFAULT 0,
    modified INTEGER NOT NULL,
    etag TEXT NOT NULL
);

-- Category index for fast filtering.
CREATE INDEX IF NOT EXISTS idx_notes_category ON notes(category);
CREATE INDEX IF NOT EXISTS idx_notes_favorite ON notes(favorite);

-- App settings table.
CREATE TABLE IF NOT EXISTS settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL DEFAULT ''
);

-- Default settings.
INSERT OR IGNORE INTO settings (key, value) VALUES ('notesPath', 'Notes');
INSERT OR IGNORE INTO settings (key, value) VALUES ('fileSuffix', '.md');
