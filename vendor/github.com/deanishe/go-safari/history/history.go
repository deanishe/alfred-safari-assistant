//
// Copyright (c) 2017 Dean Jackson <deanishe@deanishe.net>
//
// MIT Licence. See http://opensource.org/licenses/MIT
//
// Created on 2017-10-22
//

// Package history provides access to Safari's history.
//
// As the exported history was removed in High Sierra, this package
// accesses Safari's private SQLite database.
//
// The package-level functions call methods on the default History,
// which is initialised with the default Safari history database.
package history

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	// sqlite3 registers itself with sql
	_ "github.com/mattn/go-sqlite3"
)

var (
	// DefaultHistoryPath is where Safari's history database is stored.
	DefaultHistoryPath = filepath.Join(os.Getenv("HOME"), "Library/Safari/History.db")
	// MaxSearchResults is the number of results to return from a search.
	MaxSearchResults = 200
	history          *History
	// NSDate epoch starts at 00:00:00 on 1/1/2001 UTC
	tsOffset = 978307200.0
)

func init() {
	var err error
	history, err = New(DefaultHistoryPath)
	if err != nil {
		panic(err)
	}
}

// Entry is a History entry.
type Entry struct {
	Title string
	URL   string
	Time  time.Time
}

// History is a Safari history.
type History struct {
	DB *sql.DB
}

// New creates a new History from a Safari history database.
func New(filename string) (*History, error) {
	db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?mode=ro&cache=shared&_timeout=9999999&_journal=WAL", filename))
	if err != nil {
		return nil, fmt.Errorf("couldn't open database %s: %s", filename, err)
	}
	return &History{db}, nil
}

// Recent returns the specified number of most recent items from History.
// Entries without a title or with a non-HTTP* scheme are ignored.
//
// NOTE: The results will often contain many duplicates.
func Recent(count int) ([]*Entry, error) { return history.Recent(count) }
func (h *History) Recent(count int) ([]*Entry, error) {
	q := `
	SELECT url, visit_time, title
		FROM history_items
			LEFT JOIN history_visits
				ON history_visits.history_item = history_items.id
		WHERE title <> '' AND url LIKE 'http%'
		ORDER BY visit_time DESC LIMIT ?`

	return h.query(q, count)
}

// Search searches all History entries.
// Entries without a title or with a non-HTTP* scheme are ignored.
//
// The query is split into individual words, which are combined with AND:
//
//     AND title LIKE %word1% AND title LIKE %word2% etc.
//
func Search(query string) ([]*Entry, error) { return history.Search(query) }
func (h *History) Search(query string) ([]*Entry, error) {
	var (
		args []interface{}
		// Start of SQL query
		q = `
	SELECT url, visit_time, title
		FROM history_items
			LEFT JOIN history_visits
				ON history_visits.history_item = history_items.id
		WHERE title <> '' AND url LIKE 'http%'`
	)

	// Add condition and placeholder for each search term
	for _, s := range strings.Fields(query) {
		args = append(args, "%"+s+"%")
		q = q + ` AND title LIKE ?`
	}

	// Finish query
	q = q + `
		ORDER BY visit_time DESC LIMIT ?
		`

	args = append(args, MaxSearchResults)
	return h.query(q, args...)
}

// query runs an SQL query against the database.
func (h *History) query(q string, args ...interface{}) ([]*Entry, error) {
	var (
		url, title string
		when       float64
		ts         int64
		t          time.Time
		entries    []*Entry
	)
	rows, err := h.DB.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("error running query:%s with args: %+v\nerror: %s", q, args, err)
	}
	defer rows.Close()

	for rows.Next() {
		rows.Scan(&url, &when, &title)
		ts = int64(when + tsOffset)
		t = time.Unix(ts, 0).Local()
		entries = append(entries, &Entry{title, url, t})
	}

	return entries, nil
}
