package main

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
)

func TestMetadataBelongsOnlyToServer(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("GRPC_EXPLORER_DATA_DIR", dataDir)
	createLegacyDB(t, dataDir)

	if err := initDB(); err != nil {
		t.Fatalf("initDB() error = %v", err)
	}
	defer func() {
		_ = db.Close()
		db = nil
	}()

	for _, table := range []string{"saved_requests", "history"} {
		hasMeta, err := tableHasColumn(table, "meta")
		if err != nil {
			t.Fatalf("tableHasColumn(%q, meta) error = %v", table, err)
		}
		if hasMeta {
			t.Fatalf("%s should not keep legacy meta column", table)
		}
	}

	page, err := searchHistory(HistoryFilter{Limit: 10})
	if err != nil {
		t.Fatalf("searchHistory() error = %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("history item count = %d, want 1", len(page.Items))
	}
	if !strings.Contains(page.Items[0].Response, "requestMeta") {
		t.Fatalf("history debug metadata should be preserved: %s", page.Items[0].Response)
	}

	const wantMeta = `[{"key":"authorization","value":"Bearer token"}]`
	if err := updateServerMeta(1, wantMeta); err != nil {
		t.Fatalf("updateServerMeta() error = %v", err)
	}

	servers, err := searchServers("")
	if err != nil {
		t.Fatalf("searchServers() error = %v", err)
	}
	if len(servers) != 1 {
		t.Fatalf("server count = %d, want 1", len(servers))
	}
	if servers[0].Meta != wantMeta {
		t.Fatalf("server meta = %q, want %q", servers[0].Meta, wantMeta)
	}
}

func createLegacyDB(t *testing.T, dataDir string) {
	t.Helper()

	legacyDB, err := sql.Open("sqlite", filepath.Join(dataDir, "servers.db"))
	if err != nil {
		t.Fatalf("open legacy db: %v", err)
	}
	defer legacyDB.Close()

	stmts := []string{
		`CREATE TABLE servers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			url TEXT NOT NULL,
			name TEXT NOT NULL DEFAULT '',
			meta TEXT NOT NULL DEFAULT '[]',
			UNIQUE(url, name)
		)`,
		`CREATE TABLE saved_requests (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL DEFAULT '',
			server_id INTEGER NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
			method TEXT NOT NULL,
			payload TEXT NOT NULL DEFAULT '{}',
			meta TEXT NOT NULL DEFAULT '[]'
		)`,
		`CREATE TABLE history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			server_id INTEGER NOT NULL,
			method TEXT NOT NULL,
			payload TEXT NOT NULL DEFAULT '{}',
			response TEXT NOT NULL DEFAULT '{}',
			status_code INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL DEFAULT '',
			meta TEXT NOT NULL DEFAULT '[]',
			FOREIGN KEY (server_id) REFERENCES servers(id) ON DELETE CASCADE
		)`,
		`INSERT INTO servers (id, url, name, meta) VALUES (1, 'localhost:50051', 'local', '[]')`,
		`INSERT INTO saved_requests (name, server_id, method, payload, meta)
			VALUES ('saved', 1, 'hello.Greeter::SayHello', '{"name":"Ada"}', '[{"key":"x-saved","value":"1"}]')`,
		`INSERT INTO history (server_id, method, payload, response, status_code, created_at, meta)
			VALUES (
				1,
				'hello.Greeter::SayHello',
				'{"name":"Ada"}',
				'{"data":{"message":"hi"},"debug":{"requestMeta":{"authorization":["secret"]},"responseHeaders":{"x-reply":["1"]},"responseTrailers":{"grpc-status":["0"]},"statusCode":"OK"}}',
				0,
				'2026-07-03T00:00:00Z',
				'[{"key":"x-history","value":"1"}]'
			)`,
	}
	for _, stmt := range stmts {
		if _, err := legacyDB.Exec(stmt); err != nil {
			t.Fatalf("exec legacy stmt %q: %v", stmt, err)
		}
	}
}
