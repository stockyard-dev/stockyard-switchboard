package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "modernc.org/sqlite"
	"os"
	"path/filepath"
	"time"
)

type DB struct{ db *sql.DB }
type Service struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	URL       string            `json:"url"`
	Version   string            `json:"version,omitempty"`
	Tags      []string          `json:"tags"`
	Meta      map[string]string `json:"meta,omitempty"`
	Status    string            `json:"status"` // up, down, unknown
	LastPing  string            `json:"last_ping,omitempty"`
	CreatedAt string            `json:"created_at"`
}

func Open(dataDir string) (*DB, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}
	dsn := filepath.Join(dataDir, "switchboard.db") + "?_journal_mode=WAL&_busy_timeout=5000"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	for _, q := range []string{
		`CREATE TABLE IF NOT EXISTS services (id TEXT PRIMARY KEY, name TEXT NOT NULL, url TEXT DEFAULT '', version TEXT DEFAULT '', tags_json TEXT DEFAULT '[]', meta_json TEXT DEFAULT '{}', status TEXT DEFAULT 'unknown', last_ping TEXT DEFAULT '', created_at TEXT DEFAULT (datetime('now')))`,
	} {
		if _, err := db.Exec(q); err != nil {
			return nil, fmt.Errorf("migrate: %w", err)
		}
	}
	db.Exec(`CREATE TABLE IF NOT EXISTS extras(resource TEXT NOT NULL,record_id TEXT NOT NULL,data TEXT NOT NULL DEFAULT '{}',PRIMARY KEY(resource, record_id))`)
	return &DB{db: db}, nil
}
func (d *DB) Close() error { return d.db.Close() }
func genID() string        { return fmt.Sprintf("%d", time.Now().UnixNano()) }
func now() string          { return time.Now().UTC().Format(time.RFC3339) }
func (d *DB) Register(s *Service) error {
	s.ID = genID()
	s.CreatedAt = now()
	if s.Tags == nil {
		s.Tags = []string{}
	}
	if s.Meta == nil {
		s.Meta = map[string]string{}
	}
	if s.Status == "" {
		s.Status = "unknown"
	}
	tj, _ := json.Marshal(s.Tags)
	mj, _ := json.Marshal(s.Meta)
	_, err := d.db.Exec(`INSERT INTO services (id,name,url,version,tags_json,meta_json,status,created_at) VALUES (?,?,?,?,?,?,?,?)`,
		s.ID, s.Name, s.URL, s.Version, string(tj), string(mj), s.Status, s.CreatedAt)
	return err
}
func (d *DB) scan(sc interface{ Scan(...any) error }) *Service {
	var s Service
	var tj, mj string
	if err := sc.Scan(&s.ID, &s.Name, &s.URL, &s.Version, &tj, &mj, &s.Status, &s.LastPing, &s.CreatedAt); err != nil {
		return nil
	}
	json.Unmarshal([]byte(tj), &s.Tags)
	json.Unmarshal([]byte(mj), &s.Meta)
	if s.Tags == nil {
		s.Tags = []string{}
	}
	return &s
}
func (d *DB) Get(id string) *Service {
	return d.scan(d.db.QueryRow(`SELECT id,name,url,version,tags_json,meta_json,status,last_ping,created_at FROM services WHERE id=?`, id))
}
func (d *DB) List() []Service {
	rows, _ := d.db.Query(`SELECT id,name,url,version,tags_json,meta_json,status,last_ping,created_at FROM services ORDER BY name`)
	if rows == nil {
		return nil
	}
	defer rows.Close()
	var out []Service
	for rows.Next() {
		if s := d.scan(rows); s != nil {
			out = append(out, *s)
		}
	}
	return out
}
func (d *DB) Discover(tag string) []Service {
	rows, _ := d.db.Query(`SELECT id,name,url,version,tags_json,meta_json,status,last_ping,created_at FROM services WHERE tags_json LIKE ? AND status='up' ORDER BY name`, `%"`+tag+`"%`)
	if rows == nil {
		return nil
	}
	defer rows.Close()
	var out []Service
	for rows.Next() {
		if s := d.scan(rows); s != nil {
			out = append(out, *s)
		}
	}
	return out
}
func (d *DB) Heartbeat(id string) error {
	_, err := d.db.Exec(`UPDATE services SET status='up',last_ping=? WHERE id=?`, now(), id)
	return err
}
func (d *DB) Update(id string, s *Service) error {
	tj, _ := json.Marshal(s.Tags)
	mj, _ := json.Marshal(s.Meta)
	_, err := d.db.Exec(`UPDATE services SET name=?,url=?,version=?,tags_json=?,meta_json=?,status=? WHERE id=?`, s.Name, s.URL, s.Version, string(tj), string(mj), s.Status, id)
	return err
}
func (d *DB) Deregister(id string) error {
	_, err := d.db.Exec(`DELETE FROM services WHERE id=?`, id)
	return err
}

type Stats struct {
	Total int `json:"total"`
	Up    int `json:"up"`
	Down  int `json:"down"`
}

func (d *DB) Stats() Stats {
	var s Stats
	d.db.QueryRow(`SELECT COUNT(*) FROM services`).Scan(&s.Total)
	d.db.QueryRow(`SELECT COUNT(*) FROM services WHERE status='up'`).Scan(&s.Up)
	d.db.QueryRow(`SELECT COUNT(*) FROM services WHERE status='down'`).Scan(&s.Down)
	return s
}

// ─── Extras: generic key-value storage for personalization custom fields ───

func (d *DB) GetExtras(resource, recordID string) string {
	var data string
	err := d.db.QueryRow(
		`SELECT data FROM extras WHERE resource=? AND record_id=?`,
		resource, recordID,
	).Scan(&data)
	if err != nil || data == "" {
		return "{}"
	}
	return data
}

func (d *DB) SetExtras(resource, recordID, data string) error {
	if data == "" {
		data = "{}"
	}
	_, err := d.db.Exec(
		`INSERT INTO extras(resource, record_id, data) VALUES(?, ?, ?)
		 ON CONFLICT(resource, record_id) DO UPDATE SET data=excluded.data`,
		resource, recordID, data,
	)
	return err
}

func (d *DB) DeleteExtras(resource, recordID string) error {
	_, err := d.db.Exec(
		`DELETE FROM extras WHERE resource=? AND record_id=?`,
		resource, recordID,
	)
	return err
}

func (d *DB) AllExtras(resource string) map[string]string {
	out := make(map[string]string)
	rows, _ := d.db.Query(
		`SELECT record_id, data FROM extras WHERE resource=?`,
		resource,
	)
	if rows == nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var id, data string
		rows.Scan(&id, &data)
		out[id] = data
	}
	return out
}
