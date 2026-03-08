package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"unicode"

	converter "github.com/alikhil/go-convert-layout"
	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

var db *sqlx.DB

// ruLayout converts between Russian (ЙЦУКЕН) and English (QWERTY) keyboard layouts.
// Used for fuzzy search: when a user types "ghbdtn" (Russian "привет" typed on English layout),
// we also search for the converted variant.
var ruLayout, _ = converter.Create("ru")

// layoutVariant returns the keyboard-layout-converted version of a query string.
// If the query contains Cyrillic, converts to English layout; otherwise converts to Russian.
// This allows search to work regardless of which keyboard layout the user has active.
func layoutVariant(query string) string {
	hasCyrillic := false
	for _, r := range query {
		if unicode.In(r, unicode.Cyrillic) {
			hasCyrillic = true
			break
		}
	}
	if hasCyrillic {
		return ruLayout.ToEn(query)
	}
	return ruLayout.FromEn(query)
}

// dataDir returns the platform-appropriate directory for storing the SQLite database.
// Creates the directory if it doesn't exist.
func dataDir() string {
	if custom := strings.TrimSpace(os.Getenv("GRPC_EXPLORER_DATA_DIR")); custom != "" {
		if err := os.MkdirAll(custom, 0755); err != nil {
			log.Fatalf("create data dir: %v", err)
		}
		return custom
	}

	var base string
	switch runtime.GOOS {
	case "windows":
		base = os.Getenv("APPDATA")
	case "darwin":
		base = filepath.Join(os.Getenv("HOME"), "Library", "Application Support")
	default:
		base = os.Getenv("XDG_DATA_HOME")
		if base == "" {
			base = filepath.Join(os.Getenv("HOME"), ".local", "share")
		}
	}
	dir := filepath.Join(base, "pulsar-grpc")
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Fatalf("create data dir: %v", err)
	}
	return dir
}

// initDB opens the SQLite database and creates tables if they don't exist.
// Called once at startup from main().
func initDB() {
	dbPath := filepath.Join(dataDir(), "servers.db")
	log.Printf("Database: %s", dbPath)

	var err error
	db, err = sqlx.Open("sqlite", dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}

	db.MustExec(`CREATE TABLE IF NOT EXISTS servers (
		id    INTEGER PRIMARY KEY AUTOINCREMENT,
		url   TEXT NOT NULL,
		name  TEXT NOT NULL DEFAULT '',
		meta  TEXT NOT NULL DEFAULT '[]',
		UNIQUE(url, name)
	)`)

	// Migration: add meta column for databases created before this feature.
	// Intentionally ignoring the error — it fails with "duplicate column" on
	// databases that already have the column, which is expected.
	_, _ = db.Exec(`ALTER TABLE servers ADD COLUMN meta TEXT NOT NULL DEFAULT '[]'`)
	migrateServersURLUniqueness()

	db.MustExec(`CREATE TABLE IF NOT EXISTS saved_requests (
		id        INTEGER PRIMARY KEY AUTOINCREMENT,
		name      TEXT NOT NULL DEFAULT '',
		server_id INTEGER NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
		method    TEXT NOT NULL,
		payload   TEXT NOT NULL DEFAULT '{}'
	)`)

	// Migration: replace url/meta with server_id in saved_requests.
	migrateSavedRequestsToServerID()

	db.MustExec(`CREATE TABLE IF NOT EXISTS history (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		server_id   INTEGER NOT NULL,
		method      TEXT NOT NULL,
		payload     TEXT NOT NULL DEFAULT '{}',
		response    TEXT NOT NULL DEFAULT '{}',
		status_code INTEGER NOT NULL DEFAULT 0,
		created_at  TEXT NOT NULL DEFAULT '',
		FOREIGN KEY (server_id) REFERENCES servers(id) ON DELETE CASCADE
	)`)

	// SQLite requires this per-connection to enforce FK cascades.
	db.MustExec(`PRAGMA foreign_keys = ON`)
}

// migrateServersURLUniqueness replaces old UNIQUE(url) with UNIQUE(url, name).
// No-op for fresh DBs or already migrated schemas.
func migrateServersURLUniqueness() {
	type idx struct {
		Name string `db:"name"`
		Cols string `db:"cols"`
	}

	var indexes []idx
	if err := db.Select(&indexes, `
		SELECT il.name, group_concat(ii.name, ',') AS cols
		FROM pragma_index_list('servers') il
		JOIN pragma_index_info(il.name) ii
		WHERE il."unique" = 1
		GROUP BY il.name
	`); err != nil {
		return
	}

	hasURLUnique := false
	hasURLNameUnique := false
	for _, i := range indexes {
		switch i.Cols {
		case "url":
			hasURLUnique = true
		case "url,name", "name,url":
			hasURLNameUnique = true
		}
	}

	if hasURLNameUnique || !hasURLUnique {
		return
	}

	// Recreate table because SQLite cannot alter unique constraints in place.
	db.MustExec(`PRAGMA foreign_keys = OFF`)
	db.MustExec(`BEGIN`)
	db.MustExec(`CREATE TABLE servers_new (
		id    INTEGER PRIMARY KEY AUTOINCREMENT,
		url   TEXT NOT NULL,
		name  TEXT NOT NULL DEFAULT '',
		meta  TEXT NOT NULL DEFAULT '[]',
		UNIQUE(url, name)
	)`)
	db.MustExec(`INSERT INTO servers_new (id, url, name, meta)
		SELECT id, url, name, COALESCE(meta, '[]') FROM servers`)
	db.MustExec(`DROP TABLE servers`)
	db.MustExec(`ALTER TABLE servers_new RENAME TO servers`)
	db.MustExec(`COMMIT`)
}

// ─── Server CRUD ─────────────────────────────────────────────────────────────

// Server represents a saved gRPC server entry.
type Server struct {
	ID   int64  `db:"id" json:"id"`
	URL  string `db:"url" json:"url"`
	Name string `db:"name" json:"name"`
	Meta string `db:"meta" json:"meta"`
}

func searchServers(query string) ([]Server, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return queryAll[Server](`SELECT id, url, name, meta FROM servers ORDER BY id DESC`)
	}
	return searchWithLayoutVariant[Server](
		`SELECT id, url, name, meta FROM servers WHERE url LIKE ? OR name LIKE ? ORDER BY id DESC`,
		`SELECT id, url, name, meta FROM servers WHERE url LIKE ? OR name LIKE ? OR url LIKE ? OR name LIKE ? ORDER BY id DESC`,
		query,
	)
}

func createServer(url, name string) (Server, error) {
	url = strings.TrimSpace(url)
	name = strings.TrimSpace(name)

	res, err := db.Exec(`INSERT INTO servers (url, name) VALUES (?, ?)`, url, name)
	if err != nil {
		return Server{}, fmt.Errorf("insert server: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Server{}, fmt.Errorf("last insert id: %w", err)
	}
	return Server{ID: id, URL: url, Name: name, Meta: "[]"}, nil
}

func updateServerMeta(id int64, meta string) error {
	_, err := db.Exec(`UPDATE servers SET meta = ? WHERE id = ?`, meta, id)
	return err
}

func deleteServer(id int64) error {
	return deleteByID("servers", id)
}

// ─── Saved requests CRUD ─────────────────────────────────────────────────────

// SavedRequest represents a saved gRPC request with its payload.
type SavedRequest struct {
	ID       int64  `db:"id" json:"id"`
	Name     string `db:"name" json:"name"`
	ServerID int64  `db:"server_id" json:"server_id"`
	Method   string `db:"method" json:"method"`
	Payload  string `db:"payload" json:"payload"`
}

func searchSavedRequests(query, method string) ([]SavedRequest, error) {
	query = strings.TrimSpace(query)
	method = strings.TrimSpace(method)
	if query == "" && method == "" {
		return queryAll[SavedRequest](`SELECT id, name, server_id, method, payload FROM saved_requests ORDER BY id DESC`)
	}
	if query == "" {
		var items []SavedRequest
		if err := db.Select(&items, `SELECT id, name, server_id, method, payload FROM saved_requests WHERE method = ? ORDER BY id DESC`, method); err != nil {
			return nil, err
		}
		if items == nil {
			items = make([]SavedRequest, 0)
		}
		return items, nil
	}
	if method != "" {
		like := "%" + query + "%"
		alt := layoutVariant(query)
		var items []SavedRequest
		var err error
		if alt != "" && alt != query {
			likeAlt := "%" + alt + "%"
			err = db.Select(&items,
				`SELECT id, name, server_id, method, payload FROM saved_requests
				WHERE method = ? AND (name LIKE ? OR method LIKE ? OR name LIKE ? OR method LIKE ?) ORDER BY id DESC`,
				method, like, like, likeAlt, likeAlt,
			)
		} else {
			err = db.Select(&items,
				`SELECT id, name, server_id, method, payload FROM saved_requests
				WHERE method = ? AND (name LIKE ? OR method LIKE ?) ORDER BY id DESC`,
				method, like, like,
			)
		}
		if err != nil {
			return nil, err
		}
		if items == nil {
			items = make([]SavedRequest, 0)
		}
		return items, nil
	}
	return searchWithLayoutVariant[SavedRequest](
		`SELECT id, name, server_id, method, payload FROM saved_requests WHERE name LIKE ? OR method LIKE ? ORDER BY id DESC`,
		`SELECT id, name, server_id, method, payload FROM saved_requests WHERE name LIKE ? OR method LIKE ? OR name LIKE ? OR method LIKE ? ORDER BY id DESC`,
		query,
	)
}

func findSavedRequestByName(name string) (SavedRequest, error) {
	var item SavedRequest
	err := db.Get(&item, `SELECT id, name, server_id, method, payload FROM saved_requests WHERE name = ?`, name)
	return item, err
}

func updateSavedRequest(id int64, serverID int64, method, payload string) error {
	_, err := db.Exec(
		`UPDATE saved_requests SET server_id = ?, method = ?, payload = ? WHERE id = ?`,
		serverID, method, payload, id,
	)
	return err
}

func createSavedRequest(name string, serverID int64, method, payload string) (SavedRequest, error) {
	name = strings.TrimSpace(name)

	res, err := db.Exec(
		`INSERT INTO saved_requests (name, server_id, method, payload) VALUES (?, ?, ?, ?)`,
		name, serverID, method, payload,
	)
	if err != nil {
		return SavedRequest{}, fmt.Errorf("insert saved request: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return SavedRequest{}, fmt.Errorf("last insert id: %w", err)
	}
	return SavedRequest{ID: id, Name: name, ServerID: serverID, Method: method, Payload: payload}, nil
}

func deleteSavedRequest(id int64) error {
	return deleteByID("saved_requests", id)
}

// migrateSavedRequestsToServerID migrates old saved_requests with url/meta columns
// to the new schema with server_id. No-op if already migrated.
func migrateSavedRequestsToServerID() {
	// Check if old url column exists
	var cnt int
	err := db.Get(&cnt, `SELECT COUNT(*) FROM pragma_table_info('saved_requests') WHERE name = 'url'`)
	if err != nil || cnt == 0 {
		return // already migrated or error
	}

	db.MustExec(`CREATE TABLE saved_requests_new (
		id        INTEGER PRIMARY KEY AUTOINCREMENT,
		name      TEXT NOT NULL DEFAULT '',
		server_id INTEGER NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
		method    TEXT NOT NULL,
		payload   TEXT NOT NULL DEFAULT '{}'
	)`)
	db.MustExec(`INSERT INTO saved_requests_new (id, name, server_id, method, payload)
		SELECT sr.id, sr.name, s.id, sr.method, sr.payload
		FROM saved_requests sr INNER JOIN servers s ON sr.url = s.url`)
	db.MustExec(`DROP TABLE saved_requests`)
	db.MustExec(`ALTER TABLE saved_requests_new RENAME TO saved_requests`)
}

// ─── History CRUD ────────────────────────────────────────────────────────────

// HistoryEntry represents a saved gRPC request/response pair.
type HistoryEntry struct {
	ID         int64  `db:"id" json:"id"`
	ServerID   int64  `db:"server_id" json:"server_id"`
	Method     string `db:"method" json:"method"`
	Payload    string `db:"payload" json:"payload"`
	Response   string `db:"response" json:"response"`
	StatusCode int    `db:"status_code" json:"status_code"`
	CreatedAt  string `db:"created_at" json:"created_at"`
	// Joined from servers table (GET only)
	ServerURL  string `db:"server_url" json:"server_url,omitempty"`
	ServerName string `db:"server_name" json:"server_name,omitempty"`
}

const historySelectCols = `h.id, h.server_id, h.method, h.payload, h.response, h.status_code, h.created_at, s.url AS server_url, s.name AS server_name`

func searchHistory(query, method string) ([]HistoryEntry, error) {
	query = strings.TrimSpace(query)
	method = strings.TrimSpace(method)
	if query == "" && method == "" {
		var items []HistoryEntry
		if err := db.Select(&items, `SELECT `+historySelectCols+` FROM history h JOIN servers s ON s.id = h.server_id ORDER BY h.id DESC`); err != nil {
			return nil, err
		}
		if items == nil {
			items = make([]HistoryEntry, 0)
		}
		return items, nil
	}
	if query == "" {
		var items []HistoryEntry
		if err := db.Select(&items, `SELECT `+historySelectCols+` FROM history h JOIN servers s ON s.id = h.server_id WHERE h.method = ? ORDER BY h.id DESC`, method); err != nil {
			return nil, err
		}
		if items == nil {
			items = make([]HistoryEntry, 0)
		}
		return items, nil
	}

	like := "%" + query + "%"
	alt := layoutVariant(query)
	var items []HistoryEntry
	var err error
	if method != "" {
		if alt != "" && alt != query {
			likeAlt := "%" + alt + "%"
			err = db.Select(&items,
				`SELECT `+historySelectCols+` FROM history h JOIN servers s ON s.id = h.server_id
				WHERE h.method = ? AND (h.method LIKE ? OR s.url LIKE ? OR s.name LIKE ? OR h.method LIKE ? OR s.url LIKE ? OR s.name LIKE ?)
				ORDER BY h.id DESC`,
				method, like, like, like, likeAlt, likeAlt, likeAlt,
			)
		} else {
			err = db.Select(&items,
				`SELECT `+historySelectCols+` FROM history h JOIN servers s ON s.id = h.server_id
				WHERE h.method = ? AND (h.method LIKE ? OR s.url LIKE ? OR s.name LIKE ?) ORDER BY h.id DESC`,
				method, like, like, like,
			)
		}
	} else {
		if alt != "" && alt != query {
			likeAlt := "%" + alt + "%"
			err = db.Select(&items,
				`SELECT `+historySelectCols+` FROM history h JOIN servers s ON s.id = h.server_id WHERE h.method LIKE ? OR s.url LIKE ? OR s.name LIKE ? OR h.method LIKE ? OR s.url LIKE ? OR s.name LIKE ? ORDER BY h.id DESC`,
				like, like, like, likeAlt, likeAlt, likeAlt,
			)
		} else {
			err = db.Select(&items,
				`SELECT `+historySelectCols+` FROM history h JOIN servers s ON s.id = h.server_id WHERE h.method LIKE ? OR s.url LIKE ? OR s.name LIKE ? ORDER BY h.id DESC`,
				like, like, like,
			)
		}
	}
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = make([]HistoryEntry, 0)
	}
	return items, nil
}

func createHistoryEntry(serverID int64, method, payload, response string, statusCode int) (HistoryEntry, error) {
	createdAt := time.Now().UTC().Format(time.RFC3339)
	res, err := db.Exec(
		`INSERT INTO history (server_id, method, payload, response, status_code, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		serverID, method, payload, response, statusCode, createdAt,
	)
	if err != nil {
		return HistoryEntry{}, fmt.Errorf("insert history: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return HistoryEntry{}, fmt.Errorf("last insert id: %w", err)
	}
	return HistoryEntry{ID: id, ServerID: serverID, Method: method, Payload: payload, Response: response, StatusCode: statusCode, CreatedAt: createdAt}, nil
}

func deleteHistoryEntry(id int64) error {
	return deleteByID("history", id)
}

func clearHistory() error {
	_, err := db.Exec(`DELETE FROM history`)
	return err
}

// ─── Shared query helpers ────────────────────────────────────────────────────

// queryAll runs a SELECT and returns all rows, guaranteed to return an empty
// slice (not nil) when there are no results. This ensures JSON serialization
// produces [] instead of null.
func queryAll[T any](query string) ([]T, error) {
	var items []T
	if err := db.Select(&items, query); err != nil {
		return nil, err
	}
	if items == nil {
		items = make([]T, 0)
	}
	return items, nil
}

// searchWithLayoutVariant performs a LIKE search with automatic keyboard layout
// conversion. It tries both the original query and its layout-converted variant
// (e.g. Russian → English) so search works regardless of active keyboard layout.
//
// simpleQuery: SQL with 2 LIKE params (just the original query)
// extendedQuery: SQL with 4 LIKE params (original + layout variant)
func searchWithLayoutVariant[T any](simpleQuery, extendedQuery, query string) ([]T, error) {
	like := "%" + query + "%"
	alt := layoutVariant(query)

	var items []T
	var err error

	if alt != "" && alt != query {
		likeAlt := "%" + alt + "%"
		err = db.Select(&items, extendedQuery, like, like, likeAlt, likeAlt)
	} else {
		err = db.Select(&items, simpleQuery, like, like)
	}

	if err != nil {
		return nil, err
	}
	if items == nil {
		items = make([]T, 0)
	}
	return items, nil
}

// deleteByID deletes a row by ID and returns sql.ErrNoRows if nothing was deleted.
func deleteByID(table string, id int64) error {
	res, err := db.Exec(fmt.Sprintf(`DELETE FROM %s WHERE id = ?`, table), id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
