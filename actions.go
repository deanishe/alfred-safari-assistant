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

	"github.com/deanishe/awgo"
	"github.com/deanishe/go-safari"
)

var (
	tabActions map[string]TabActionable
	urlActions map[string]URLActionable
	// Extensions of files that need to be run via /usr/bin/osascript
	osaExts = map[string]bool{
		".scpt":        true,
		".js":          true,
		".applescript": true,
		".scptd":       true,
	}
	iconExts = []string{".png", ".icns", ".jpg", ".jpeg", ".gif"}
)

func init() {
	tabActions = map[string]TabActionable{}
	urlActions = map[string]URLActionable{}
	// Built-in actions
	for _, a := range []Actionable{
		&closeTab{},
		&closeTabsLeft{},
		&closeTabsRight{},
		&closeTabsOther{},
		&closeWindow{},
		&openURLAction{},
	} {
		if err := Register(a); err != nil {
			panic(err)
		}
	}
}

// Register an action.
func Register(action Actionable) error {
	if a, ok := action.(TabActionable); ok {
		tabActions[a.Title()] = a
		return nil
	}
	if a, ok := action.(URLActionable); ok {
		urlActions[a.Title()] = a
		return nil
	}
	return fmt.Errorf("Unknown action type : %+v", action)
}

// URLActions returns registered URLActions
func URLActions() []URLActionable {
	var (
		i    int
		acts = make([]URLActionable, len(urlActions))
	)
	for _, a := range urlActions {
		acts[i] = a
		i++
	}
	return acts
}

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
func TabActions() []TabActionable {
	var (
		i    int
		acts = make([]TabActionable, len(tabActions))
	)
	for _, a := range tabActions {
		acts[i] = a
		i++
	}
	return acts
}

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
	Icon() *aw.Icon
}

// TabActionable is an action that can be performed on a tab.
type TabActionable interface {
	Actionable
	Run(t *safari.Tab) error
}

// URLActionable is an action that can be performed on a URL.
type URLActionable interface {
	Actionable
	Run(url *url.URL) error
}

type baseTabAction struct{}

func (a *baseTabAction) Icon() *aw.Icon { return IconTab }

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

func (a *baseURLAction) Icon() *aw.Icon { return IconURL }

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
func getIcon(p, typ string) *aw.Icon {
	x := filepath.Ext(p)
	r := p[0 : len(p)-len(x)]
	for _, x := range iconExts {
		ip := r + x
		if _, err := os.Stat(ip); err == nil {
			return &aw.Icon{Value: ip}
		}
	}
	switch typ {
	case "url":
		return IconURL
	case "tab":
		return IconTab
	default:
		return IconDefault
	}
}

// script wraps a script's path and icon details.
type script struct {
	Path string
	Icon *aw.Icon
}

// newScript initialises a new script.
func newScript(p, typ string) *script {
	i := getIcon(p, typ)
	return &script{p, i}
}

// scriptRunner is a base struct for running scripts.
type scriptRunner struct {
	Script *script
}

// Implement Actionable.
func (a *scriptRunner) Title() string {
	return scriptTitle(a.Script.Path)
}
func (a *scriptRunner) Icon() *aw.Icon { return a.Script.Icon }

// run runs a command line script.
func (a *scriptRunner) run(args ...string) error {
	var cmd *exec.Cmd
	sargs := []string{}
	if isExecutable(a.Script.Path) {
		cmd = exec.Command(a.Script.Path, args...)
	} else if isOSAScript(a.Script.Path) {
		ext := filepath.Ext(a.Script.Path)
		if ext == ".js" {
			sargs = append(sargs, "-l", "JavaScript")
		}
		sargs = append(sargs, a.Script.Path)
		sargs = append(sargs, args...)
		cmd = exec.Command("/usr/bin/osascript", sargs...)
	} else {
		return fmt.Errorf("Don't know how to run script: %s", a.Script.Path)
	}
	log.Printf("%v", cmd)
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

			typ := filepath.Base(dp)
			if isOSAScript(p) || isExecutable(p) {
				// Type is based on parent directory
				s := newScript(p, typ)
				switch typ {
				case "tab":
					Register(&tabRunner{scriptRunner{s}})
					log.Printf("Tab Script `%s` from `%s`", scriptTitle(p), p)
				case "url":
					Register(&urlRunner{scriptRunner{s}})
					log.Printf("URL Script `%s` from `%s`", scriptTitle(p), p)
					// default:
					// 	log.Printf("I (%s) : %s", filepath.Base(dp))
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

// scriptTitle returns the basename of a path without the extension.
func scriptTitle(p string) string {
	fn := filepath.Base(p)
	return fn[0 : len(fn)-len(filepath.Ext(fn))]
}

// isOSAScript returns true if script should be run with /usr/bin/osascript.
func isOSAScript(p string) bool {
	x := filepath.Ext(p)
	return osaExts[x] == true
}

// isExecutable return true if the executable bit is set on script.
func isExecutable(p string) bool {
	info, err := os.Stat(p)
	if err != nil {
		log.Printf("Couldn't stat file (%s): %s", err, p)
		return false
	}
	perms := uint32(info.Mode().Perm())
	return perms&0111 != 0
}
