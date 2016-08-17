//
// Copyright (c) 2016 Dean Jackson <deanishe@deanishe.net>
//
// MIT Licence. See http://opensource.org/licenses/MIT
//
// Created on 2016-07-30
//

package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"

	"gogs.deanishe.net/deanishe/awgo"
	"gogs.deanishe.net/deanishe/go-safari"
)

var (
	tabActions []TabActionable
	urlActions []URLActionable
	scriptExts = []string{".scpt", ".sh", ".bash", ".zsh"}
	iconExts   = []string{".png", ".icns", ".jpg", ".jpeg", ".gif"}
)

func init() {
	tabActions = []TabActionable{
		&closeTab{},
		&closeTabsLeft{},
		&closeTabsRight{},
		&closeTabsOther{},
		&closeWindow{},
	}
	urlActions = []URLActionable{
		&openURLAction{},
	}
}

// Register an action.
func Register(action Actionable) error {
	if a, ok := action.(TabActionable); ok {
		tabActions = append(tabActions, a)
		return nil
	}
	if a, ok := action.(URLActionable); ok {
		urlActions = append(urlActions, a)
		return nil
	}
	return fmt.Errorf("Unknown action type : %+v", action)
}

// URLActions returns registered URLActions
func URLActions() []URLActionable { return urlActions }

// URLAction returns the first registered URLActionable with the matching title.
func URLAction(title string) URLActionable {
	acts := URLActions()
	for _, a := range acts {
		if a.Title() == title {
			return a
		}
	}
	return nil
}

// TabActions returns registered TabActions
func TabActions() []TabActionable { return tabActions }

// TabAction returns the first registered TabActionable with the matching title.
func TabAction(title string) TabActionable {
	acts := TabActions()
	for _, a := range acts {
		if a.Title() == title {
			return a
		}
	}
	return nil
}

// Actionable is a base interface for tab/URL actions.
type Actionable interface {
	Title() string
	Icon() string
	IconType() string
}

// TabActionable is an action that can be performed on a tab.
type TabActionable interface {
	Actionable
	Run(t *safari.Tab) error
}

// URLActionable is an action that can be performed on a URL.
type URLActionable interface {
	Actionable
	Run(u *url.URL) error
}

type baseTabAction struct{}

func (a *baseTabAction) Icon() string     { return "tab.png" }
func (a *baseTabAction) IconType() string { return "" }

type closeTab struct {
	baseTabAction
}

// Implement Actionable.
func (a *closeTab) Title() string           { return "Close Tab" }
func (a *closeTab) Run(t *safari.Tab) error { return safari.CloseTab(t.WindowIndex, t.Index) }

type closeTabsOther struct {
	baseTabAction
}

// Implement Actionable.
func (a *closeTabsOther) Title() string { return "Close Other Tabs" }
func (a *closeTabsOther) Run(t *safari.Tab) error {
	return safari.CloseTabsOther(t.WindowIndex, t.Index)
}

type closeTabsLeft struct {
	baseTabAction
}

// Implement Actionable.
func (a *closeTabsLeft) Title() string { return "Close Tabs to Left" }
func (a *closeTabsLeft) Run(t *safari.Tab) error {
	return safari.CloseTabsLeft(t.WindowIndex, t.Index)
}

type closeTabsRight struct {
	baseTabAction
}

// Implement Actionable.
func (a *closeTabsRight) Title() string { return "Close Tabs to Right" }
func (a *closeTabsRight) Run(t *safari.Tab) error {
	return safari.CloseTabsRight(t.WindowIndex, t.Index)
}

type closeWindow struct {
	baseTabAction
}

// Implement Actionable.
func (a *closeWindow) Title() string           { return "Close Window" }
func (a *closeWindow) Run(t *safari.Tab) error { return safari.CloseWin(t.WindowIndex) }

type baseURLAction struct{}

func (a *baseURLAction) Icon() string     { return workflow.IconWeb.Value }
func (a *baseURLAction) IconType() string { return workflow.IconWeb.Type }

type openURLAction struct {
	baseURLAction
}

// Implement Actionable.
func (a *openURLAction) Title() string { return "Open URL in Default Browser" }
func (a *openURLAction) Run(u *url.URL) error {
	cmd := exec.Command("/usr/bin/open", u.String())
	return cmd.Run()
}

// getIcon returns icon and icon type for script path. It looks for
// an icon file sharing the same basename (i.e. only the extension differs).
// Otherwise it returns the script's own icon.
func getIcon(p string) (icon, iconType string) {
	x := filepath.Ext(p)
	r := p[0 : len(p)-len(x)]
	for _, x := range iconExts {
		ip := r + x
		if _, err := os.Stat(ip); err == nil {
			return ip, ""
		}
	}
	return p, "fileicon"
}

// script wraps a script's path and icon details.
type script struct {
	Path     string
	Icon     string
	IconType string
}

// newScript initialises a new script.
func newScript(p string) *script {
	i, t := getIcon(p)
	return &script{p, i, t}
}

// scriptRunner is a base struct for running scripts.
type scriptRunner struct {
	Script *script
}

// Implement Actionable.
func (a *scriptRunner) Title() string {
	fn := filepath.Base(a.Script.Path)
	return fn[0 : len(fn)-len(filepath.Ext(fn))-1]
}
func (a *scriptRunner) Icon() string     { return a.Script.Icon }
func (a *scriptRunner) IconType() string { return a.Script.IconType }

// run runs a command line script.
func (a *scriptRunner) run(args ...string) error {
	cmd := exec.Command(a.Script.Path, args...)
	return cmd.Run()
}

// tabRunner executes a tab script.
type tabRunner struct {
	scriptRunner
}

// Run implements TabActionable.
func (a *tabRunner) Run(t *safari.Tab) error {
	return a.run(fmt.Sprintf("%d", t.WindowIndex), fmt.Sprintf("%d", t.Index))
}

// tabRunner executes a URL script.
type urlRunner struct {
	scriptRunner
}

// Run implements URLActionable.
func (a *urlRunner) Run(u *url.URL) error {
	return a.run(u.String())
}

// LoadScripts finds scripts in directories and registers them.
func LoadScripts(dirs ...string) error {
	errs := []error{}

	for _, dp := range dirs {

		err := filepath.Walk(dp, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}

			ext := filepath.Ext(p)
			for _, x := range scriptExts {
				if ext == x {
					s := newScript(p)
					log.Printf("Script : %v", s)

					// Type is based on parent directory
					switch filepath.Base(dp) {

					case "tab":
						Register(&tabRunner{scriptRunner{s}})
					case "url":
						Register(&urlRunner{scriptRunner{s}})
					default:
						log.Printf("Invalid action type : %s", filepath.Base(dp))

					}
				}
			}
			return nil
		})

		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}
