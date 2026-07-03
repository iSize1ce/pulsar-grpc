package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

var db *sqlx.DB

// allowedTables is the strict set of tables that deleteByID may operate on.
// This guards against accidental misuse since the table name is interpolated into SQL.
var allowedTables = map[string]struct{}{
	"servers":        {},
	"saved_requests": {},
	"history":        {},
}

// dataDir returns the platform-appropriate directory for storing the SQLite database.
// Creates the directory if it doesn't exist.
func dataDir() (string, error) {
	if custom := strings.TrimSpace(os.Getenv("GRPC_EXPLORER_DATA_DIR")); custom != "" {
		if err := os.MkdirAll(custom, 0755); err != nil {
			return "", fmt.Errorf("create data dir %q: %w", custom, err)
		}
		return custom, nil
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
		return "", fmt.Errorf("create data dir %q: %w", dir, err)
	}
	return dir, nil
}

// initDB opens the SQLite database and ensures all tables exist.
// Called once at startup from main().
func initDB() error {
	dir, err := dataDir()
	if err != nil {
		return err
	}

	dbPath := filepath.Join(dir, "servers.db")
	slog.Info("database", "path", dbPath)

	db, err = sqlx.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("open db %q: %w", dbPath, err)
	}

	stmts := []string{
		`CREATE TABLE IF NOT EXISTS servers (
			id    INTEGER PRIMARY KEY AUTOINCREMENT,
			url   TEXT NOT NULL,
			name  TEXT NOT NULL DEFAULT '',
			meta  TEXT NOT NULL DEFAULT '[]',
			UNIQUE(url, name)
		)`,
		`CREATE TABLE IF NOT EXISTS saved_requests (
			id        INTEGER PRIMARY KEY AUTOINCREMENT,
			name      TEXT NOT NULL DEFAULT '',
			server_id INTEGER NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
			method    TEXT NOT NULL,
			payload   TEXT NOT NULL DEFAULT '{}'
		)`,
		`CREATE TABLE IF NOT EXISTS history (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			server_id   INTEGER NOT NULL,
			method      TEXT NOT NULL,
			payload     TEXT NOT NULL DEFAULT '{}',
			response    TEXT NOT NULL DEFAULT '{}',
			status_code INTEGER NOT NULL DEFAULT 0,
			created_at  TEXT NOT NULL DEFAULT '',
			FOREIGN KEY (server_id) REFERENCES servers(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_history_server_id ON history(server_id)`,
		`CREATE INDEX IF NOT EXISTS idx_history_method ON history(method)`,
		`CREATE INDEX IF NOT EXISTS idx_history_status_code ON history(status_code)`,
		// SQLite requires this per-connection to enforce FK cascades.
		`PRAGMA foreign_keys = ON`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return fmt.Errorf("initDB exec %q: %w", s[:min(40, len(s))], err)
		}
	}
	if err := dropLegacyMetaColumns(); err != nil {
		return err
	}
	return nil
}

// min returns the smaller of a and b.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func dropLegacyMetaColumns() error {
	for _, table := range []string{"saved_requests", "history"} {
		if err := dropColumnIfExists(table, "meta"); err != nil {
			return err
		}
	}
	return nil
}

func dropColumnIfExists(table, column string) error {
	exists, err := tableHasColumn(table, column)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	if _, err := db.Exec(fmt.Sprintf(`ALTER TABLE %s DROP COLUMN %s`, table, column)); err != nil {
		return fmt.Errorf("drop legacy column %s.%s: %w", table, column, err)
	}
	return nil
}

func tableHasColumn(table, column string) (bool, error) {
	if _, ok := allowedTables[table]; !ok {
		return false, fmt.Errorf("tableHasColumn: unknown table %q", table)
	}

	rows, err := db.Queryx(fmt.Sprintf(`PRAGMA table_info(%s)`, table))
	if err != nil {
		return false, fmt.Errorf("inspect table %s: %w", table, err)
	}
	defer rows.Close()

	for rows.Next() {
		var cid, notNull, pk int
		var name, columnType string
		var defaultValue sql.NullString
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &pk); err != nil {
			return false, fmt.Errorf("scan table_info %s: %w", table, err)
		}
		if name == column {
			return true, nil
		}
	}
	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("iterate table_info %s: %w", table, err)
	}
	return false, nil
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
		return queryAll[Server](db, `SELECT id, url, name, meta FROM servers ORDER BY id DESC`)
	}
	return searchWithLayoutVariant[Server](db,
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
		return Server{}, fmt.Errorf("insert server last id: %w", err)
	}
	return Server{ID: id, URL: url, Name: name, Meta: "[]"}, nil
}

func updateServerMeta(id int64, meta string) error {
	_, err := db.Exec(`UPDATE servers SET meta = ? WHERE id = ?`, meta, id)
	if err != nil {
		return fmt.Errorf("update server meta id=%d: %w", id, err)
	}
	return nil
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
		return queryAll[SavedRequest](db, `SELECT id, name, server_id, method, payload FROM saved_requests ORDER BY id DESC`)
	}

	if query == "" {
		return queryAll[SavedRequest](db,
			`SELECT id, name, server_id, method, payload FROM saved_requests WHERE method = ? ORDER BY id DESC`,
			method,
		)
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
			return nil, fmt.Errorf("search saved_requests: %w", err)
		}
		if items == nil {
			items = make([]SavedRequest, 0)
		}
		return items, nil
	}

	return searchWithLayoutVariant[SavedRequest](db,
		`SELECT id, name, server_id, method, payload FROM saved_requests WHERE name LIKE ? OR method LIKE ? ORDER BY id DESC`,
		`SELECT id, name, server_id, method, payload FROM saved_requests WHERE name LIKE ? OR method LIKE ? OR name LIKE ? OR method LIKE ? ORDER BY id DESC`,
		query,
	)
}

func updateSavedRequest(id int64, serverID int64, method, payload string) error {
	_, err := db.Exec(
		`UPDATE saved_requests SET server_id = ?, method = ?, payload = ? WHERE id = ?`,
		serverID, method, payload, id,
	)
	if err != nil {
		return fmt.Errorf("update saved_request id=%d: %w", id, err)
	}
	return nil
}

func createSavedRequest(name string, serverID int64, method, payload string) (SavedRequest, error) {
	name = strings.TrimSpace(name)

	res, err := db.Exec(
		`INSERT INTO saved_requests (name, server_id, method, payload) VALUES (?, ?, ?, ?)`,
		name, serverID, method, payload,
	)
	if err != nil {
		return SavedRequest{}, fmt.Errorf("insert saved_request: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return SavedRequest{}, fmt.Errorf("insert saved_request last id: %w", err)
	}
	return SavedRequest{ID: id, Name: name, ServerID: serverID, Method: method, Payload: payload}, nil
}

func deleteSavedRequest(id int64) error {
	return deleteByID("saved_requests", id)
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
	// Joined from servers table (GET only).
	ServerURL  string `db:"server_url" json:"server_url,omitempty"`
	ServerName string `db:"server_name" json:"server_name,omitempty"`
}

const historySelectCols = `h.id, h.server_id, h.method, h.payload, h.response, h.status_code, h.created_at, s.url AS server_url, s.name AS server_name`

type HistoryFilter struct {
	Query    string
	Method   string
	ServerID int64
	Status   string
	Limit    int
	Offset   int
}

type HistoryPage struct {
	Items  []HistoryEntry `json:"items"`
	Total  int            `json:"total"`
	Limit  int            `json:"limit"`
	Offset int            `json:"offset"`
}

func searchHistory(filter HistoryFilter) (HistoryPage, error) {
	filter.Query = strings.TrimSpace(filter.Query)
	filter.Method = strings.TrimSpace(filter.Method)
	filter.Status = strings.TrimSpace(filter.Status)
	filter.Limit = clampHistoryLimit(filter.Limit)
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	baseFrom := ` FROM history h JOIN servers s ON s.id = h.server_id`
	whereParts := make([]string, 0, 4)
	args := make([]any, 0, 10)

	if filter.Method != "" {
		whereParts = append(whereParts, `h.method = ?`)
		args = append(args, filter.Method)
	}
	if filter.ServerID > 0 {
		whereParts = append(whereParts, `h.server_id = ?`)
		args = append(args, filter.ServerID)
	}
	switch filter.Status {
	case "ok":
		whereParts = append(whereParts, `h.status_code = 0`)
	case "error":
		whereParts = append(whereParts, `h.status_code != 0`)
	}
	if filter.Query != "" {
		searchParts := []string{`h.method LIKE ?`, `s.url LIKE ?`, `s.name LIKE ?`}
		like := "%" + filter.Query + "%"
		args = append(args, like, like, like)

		if alt := layoutVariant(filter.Query); alt != "" && alt != filter.Query {
			likeAlt := "%" + alt + "%"
			searchParts = append(searchParts, `h.method LIKE ?`, `s.url LIKE ?`, `s.name LIKE ?`)
			args = append(args, likeAlt, likeAlt, likeAlt)
		}
		whereParts = append(whereParts, `(`+strings.Join(searchParts, ` OR `)+`)`)
	}

	whereSQL := ""
	if len(whereParts) > 0 {
		whereSQL = ` WHERE ` + strings.Join(whereParts, ` AND `)
	}

	var total int
	if err := db.Get(&total, `SELECT COUNT(*)`+baseFrom+whereSQL, args...); err != nil {
		return HistoryPage{}, fmt.Errorf("count history: %w", err)
	}

	queryArgs := append(append([]any{}, args...), filter.Limit, filter.Offset)
	items := make([]HistoryEntry, 0)
	if err := db.Select(
		&items,
		`SELECT `+historySelectCols+baseFrom+whereSQL+` ORDER BY h.id DESC LIMIT ? OFFSET ?`,
		queryArgs...,
	); err != nil {
		return HistoryPage{}, fmt.Errorf("search history: %w", err)
	}

	return HistoryPage{
		Items:  items,
		Total:  total,
		Limit:  filter.Limit,
		Offset: filter.Offset,
	}, nil
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
		return HistoryEntry{}, fmt.Errorf("insert history last id: %w", err)
	}
	return HistoryEntry{
		ID: id, ServerID: serverID, Method: method,
		Payload: payload, Response: response,
		StatusCode: statusCode, CreatedAt: createdAt,
	}, nil
}

func deleteHistoryEntry(id int64) error {
	return deleteByID("history", id)
}

func clearHistory() error {
	if _, err := db.Exec(`DELETE FROM history`); err != nil {
		return fmt.Errorf("clear history: %w", err)
	}
	return nil
}

func clampHistoryLimit(limit int) int {
	switch {
	case limit <= 0:
		return 50
	case limit > 200:
		return 200
	default:
		return limit
	}
}

// ─── Shared query helpers ────────────────────────────────────────────────────

// queryAll runs a SELECT and returns all rows, guaranteed to return an empty
// slice (not nil) when there are no results. This ensures JSON serialization
// produces [] instead of null.
func queryAll[T any](db *sqlx.DB, query string, args ...any) ([]T, error) {
	var items []T
	if err := db.Select(&items, query, args...); err != nil {
		return nil, err
	}
	if items == nil {
		items = make([]T, 0)
	}
	return items, nil
}

// deleteByID deletes a row by ID and returns sql.ErrNoRows if nothing was deleted.
func deleteByID(table string, id int64) error {
	if _, ok := allowedTables[table]; !ok {
		return fmt.Errorf("deleteByID: unknown table %q", table)
	}
	res, err := db.Exec(fmt.Sprintf(`DELETE FROM %s WHERE id = ?`, table), id)
	if err != nil {
		return fmt.Errorf("delete from %s id=%d: %w", table, id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete from %s rows affected: %w", table, err)
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
