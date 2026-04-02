package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type DB struct { db *sql.DB }

type Service struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Host         string   `json:"host"`
	Port         int      `json:"port"`
	Status       string   `json:"status"`
	Tags         string   `json:"tags"`
	CreatedAt    string   `json:"created_at"`
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
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS services (
			id TEXT PRIMARY KEY,\n\t\t\tname TEXT DEFAULT '',\n\t\t\thost TEXT DEFAULT '',\n\t\t\tport INTEGER DEFAULT 0,\n\t\t\tstatus TEXT DEFAULT 'up',\n\t\t\ttags TEXT DEFAULT '',
			created_at TEXT DEFAULT (datetime('now'))
		)`)
	if err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return &DB{db: db}, nil
}

func (d *DB) Close() error { return d.db.Close() }

func genID() string { return fmt.Sprintf("%d", time.Now().UnixNano()) }

func (d *DB) Create(e *Service) error {
	e.ID = genID()
	e.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	_, err := d.db.Exec(`INSERT INTO services (id, name, host, port, status, tags, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		e.ID, e.Name, e.Host, e.Port, e.Status, e.Tags, e.CreatedAt)
	return err
}

func (d *DB) Get(id string) *Service {
	row := d.db.QueryRow(`SELECT id, name, host, port, status, tags, created_at FROM services WHERE id=?`, id)
	var e Service
	if err := row.Scan(&e.ID, &e.Name, &e.Host, &e.Port, &e.Status, &e.Tags, &e.CreatedAt); err != nil {
		return nil
	}
	return &e
}

func (d *DB) List() []Service {
	rows, err := d.db.Query(`SELECT id, name, host, port, status, tags, created_at FROM services ORDER BY created_at DESC`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var result []Service
	for rows.Next() {
		var e Service
		if err := rows.Scan(&e.ID, &e.Name, &e.Host, &e.Port, &e.Status, &e.Tags, &e.CreatedAt); err != nil {
			continue
		}
		result = append(result, e)
	}
	return result
}

func (d *DB) Delete(id string) error {
	_, err := d.db.Exec(`DELETE FROM services WHERE id=?`, id)
	return err
}

func (d *DB) Count() int {
	var n int
	d.db.QueryRow(`SELECT COUNT(*) FROM services`).Scan(&n)
	return n
}
