//
// Copyright (c) 2016 Dean Jackson <deanishe@deanishe.net>
//
// MIT Licence. See http://opensource.org/licenses/MIT
//
// Created on 2016-07-30
//

package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	aw "github.com/deanishe/awgo"
	"github.com/deanishe/awgo/util"
	safari "github.com/deanishe/go-safari"
)

var (
	blacklist         = map[string]bool{} // Names of actions user has deactivated
	blacklistFilename = "blacklist.txt"
	blacklistTemplate = `#
# Action blacklist
# ----------------
#
# Add the names of action scripts to ignore to this file,
# one per line without any file extensions, e.g. add
#
# Open in Firefox
#
# to prevent the "Open in Firefox" action from being shown in
# any list of actions.
#
# NOTE: This doesn't prevent an action from being called: it only
# prevents it from being shown in any action lists in Alfred's UI.
# That means you can blacklist actions here that you've assigned
# as alterate actions via the workflow configuration sheet.
#
# Empty lines and lines beginning with # are ignored.
#

`
	tabActions = map[string]TabActionable{}
	urlActions = map[string]URLActionable{}
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

// doBlacklist adds the specified action names to the blacklist.
func doBlacklist() error {
	wf.Configure(aw.TextErrors(true))
	return addToBlacklist(scriptNames...)
}

// Blacklist instructs registry to ignore scripts with the given name.
// This only applies to lists of all actions; if an action is requested
// by name, e.g. via URLAction(), it will still be returned.
func Blacklist(actionName string) { blacklist[actionName] = true }

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
	return fmt.Errorf("unknown action type : %+v", action)
}

// URLActions returns registered URLActions
func URLActions() []URLActionable {
	acts := []URLActionable{}
	for _, a := range urlActions {
		if !blacklist[a.Title()] {
			acts = append(acts, a)
		}
	}
	return acts
}

// URLAction returns the first registered URLActionable with the matching title.
func URLAction(title string) URLActionable {
	for _, a := range URLActions() {
		if a.Title() == title {
			return a
		}
	}
	return nil
}

// TabActions returns registered TabActions
func TabActions() []TabActionable {
	acts := []TabActionable{}
	for _, a := range tabActions {
		if !blacklist[a.Title()] {
			acts = append(acts, a)
		}
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
func (a *openURLAction) Title() string { return "Open in Default Browser" }
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
	if err := loadBlacklist(); err != nil {
		return err
	}

	errs := []error{}

	for _, dp := range dirs {
		util.MustExist(dp)
		err := filepath.Walk(dp, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}

			typ := filepath.Base(dp) // Type is based on parent directory
			if isOSAScript(p) || isExecutable(p) {
				s := newScript(p, typ)
				switch typ {
				case "tab":
					r := &tabRunner{scriptRunner{s}}
					if blacklist[r.Title()] {
						log.Printf("blacklisted: %s", r.Title())
					} else {
						if err := Register(r); err != nil {
							log.Println(err)
						} else {
							log.Printf("Tab Script %q from %q", scriptTitle(p), p)
						}
					}

				case "url":
					r := &urlRunner{scriptRunner{s}}
					if blacklist[r.Title()] {
						log.Printf("blacklisted: %s", r.Title())

					} else {
						if err := Register(r); err != nil {
							log.Println(err)
						} else {
							log.Printf("URL Script %q from %q", scriptTitle(p), p)
						}
					}
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

// Return path to initialised blacklist.
func initBlacklist() (string, error) {
	path := filepath.Join(wf.DataDir(), blacklistFilename)
	if !util.PathExists(path) {
		if err := ioutil.WriteFile(path, []byte(blacklistTemplate), 0600); err != nil {
			return "", err
		}
	}
	return path, nil
}

// Append given script names to blacklist.
func addToBlacklist(names ...string) error {
	path, err := initBlacklist()
	if err != nil {
		return err
	}

	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, s := range names {
		if _, err := file.WriteString(s + "\n"); err != nil {
			return err
		}
	}
	return nil
}

// loadBlacklist loads user-blacklisted action names.
func loadBlacklist() error {
	path, err := initBlacklist()
	if err != nil {
		return err
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}

	// Treat each non-empty line that doesn't start with # as a script name
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		Blacklist(line)
	}
	return scanner.Err()
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
