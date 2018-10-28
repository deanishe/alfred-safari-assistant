// Copyright (c) 2018 Dean Jackson <deanishe@deanishe.net>
// MIT Licence applies http://opensource.org/licenses/MIT

// Package cloud provides access to Safari's iCloud Tabs.
package cloud

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	// sqlite3 registers itself with sql
	_ "github.com/mattn/go-sqlite3"
)

var (
	// DefaultTabsPath is the path to the default CloudTabs database.
	DefaultTabsPath = filepath.Join(os.Getenv("HOME"), "Library/Safari/CloudTabs.db")
	hostname        string
	tabs            *CloudTabs
)

func init() {
	var (
		data []byte
		err  error
	)
	tabs, err = New(DefaultTabsPath)
	if err != nil {
		panic(err)
	}
	data, err = exec.Command("/usr/sbin/scutil", "--get", "ComputerName").Output()
	if err != nil {
		panic(err)
	}
	hostname = strings.TrimSpace(string(data))
}

// CloudTabs is a collection of Tabs.
type CloudTabs struct {
	DB *sql.DB
}

// New creates a new Tabs from a Safari CloudTabs.db database.
func New(filename string) (*CloudTabs, error) {
	db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?mode=ro&cache=shared&_timeout=9999999&_journal=WAL", filename))
	if err != nil {
		return nil, fmt.Errorf("couldn't open database %s: %s", filename, err)
	}
	return &CloudTabs{db}, nil
}

// Tabs returns all Cloud Tabs. Tabs for the current device are ignored.
func Tabs() ([]*Tab, error) { return tabs.Tabs() }

// Tabs returns all Cloud Tabs. Tabs for the current device are ignored.
func (c *CloudTabs) Tabs() ([]*Tab, error) {
	var (
		q = `
		SELECT t.title, t.url, d.device_name
			FROM cloud_tabs t
				LEFT JOIN cloud_tab_devices d
					ON t.device_uuid = d.device_uuid
		WHERE d.device_name != ?
		`
		title, url, device string
		tab                *Tab
		tabs               []*Tab
	)

	rows, err := c.DB.Query(q, hostname)
	if err != nil {
		return nil, fmt.Errorf("error running query:%s error: %s", q, err)
	}
	defer rows.Close()

	for rows.Next() {
		rows.Scan(&title, &url, &device)
		tab = &Tab{title, url, device}
		tabs = append(tabs, tab)
	}

	return tabs, nil
}

// Tab is a cloud tab.
type Tab struct {
	Title  string // Tab title
	URL    string // URL
	Device string // Computer/phone/tablet name
}
