//
// Copyright (c) 2016 Dean Jackson <deanishe@deanishe.net>
//
// MIT Licence. See http://opensource.org/licenses/MIT
//
// Created on 2016-05-29
//

package safari

import (
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/juju/deputy"
)

// Tab is a Safari tab.
type Tab struct {
	Index       int
	WindowIndex int
	Title       string
	URL         string
	Active      bool
}

// RunJS executes JavaScript in this tab.
func (t *Tab) RunJS(js string) error {
	_, err := runJXA(jsRunJavaScript, fmt.Sprintf("%d", t.WindowIndex),
		fmt.Sprintf("%d", t.Index), js)
	return err
}

// Activate activates this tab.
func (t *Tab) Activate() error {
	if t.Active {
		return nil
	}
	return Activate(t.WindowIndex, t.Index)
}

// Window is a Safari window.
type Window struct {
	Index     int
	ActiveTab int
	Tabs      []*Tab
}

// Windows returns information about Safari's open windows.
//
// NOTE: This function takes a long time (~0.5 seconds) to complete as
// it calls Safari via the Scripting Bridge, which is slow as shit.
//
// You would be wise to cache these data for a few seconds.
func Windows() ([]*Window, error) {
	wins := []*Window{}

	if err := runJXA2JSON(jsGetTabs, &wins); err != nil {
		return nil, err
	}
	return wins, nil
}

// ActiveTab returns information about Safari's active tab.
//
// NOTE: This function calls Safari via the Scripting Bridge, so it's
// quite slow.
func ActiveTab() (*Tab, error) {
	tab := &Tab{}

	if err := runJXA2JSON(jsGetCurrentTab, &tab); err != nil {
		return nil, err
	}
	return tab, nil
}

// Activate activates the specified Safari window (and tab). If tab is 0,
// the active tab will not be changed.
func Activate(win, tab int) error {

	args := []string{fmt.Sprintf("%d", win)}
	if tab > 0 {
		args = append(args, fmt.Sprintf("%d", tab))
	}

	if _, err := runJXA(jsActivate, args...); err != nil {
		return err
	}
	return nil
}

// ActivateTab activates the specified tab.
func ActivateTab(win, tab int) error {
	return Activate(win, tab)
}

// ActivateWin activates the specified window.
func ActivateWin(win int) error {
	return Activate(win, 0)
}

// closeStuff runs script jsClose with the given arguments.
func closeStuff(what string, win, tab int) error {

	if win == 0 { // Default to frontmost window
		win = 1
	}
	args := []string{what, fmt.Sprintf("%d", win)}

	if tab > 0 {
		args = append(args, fmt.Sprintf("%d", tab))
	}

	if _, err := runJXA(jsClose, args...); err != nil {
		return err
	}

	return nil
}

// Close closes the specified tab.
// If win is 0, the frontmost window is assumed. If tab is 0, current tab is
// assumed.
func Close(win, tab int) error { return closeStuff("tab", win, tab) }

// CloseWin closes the specified window. If win is 0, the frontmost window is closed.
func CloseWin(win int) error { return closeStuff("win", win, 0) }

// CloseTab closes the specified tab. If win is 0, frontmost window is assumed.
// If tab is 0, current tab is closed.
func CloseTab(win, tab int) error { return closeStuff("tab", win, tab) }

// CloseTabsOther closes all other tabs in win.
func CloseTabsOther(win, tab int) error { return closeStuff("tabs-other", win, tab) }

// CloseTabsLeft closes tabs to the left of the specified one.
func CloseTabsLeft(win, tab int) error { return closeStuff("tabs-left", win, tab) }

// CloseTabsRight closes tabs to the right of the specified one.
func CloseTabsRight(win, tab int) error { return closeStuff("tabs-right", win, tab) }

// runJXA executes JavaScript script with /usr/bin/osascript and returns the
// script's output on STDOUT.
func runJXA(script string, argv ...string) ([]byte, error) {

	data := []byte{}

	d := deputy.Deputy{
		Errors:    deputy.FromStderr,
		StdoutLog: func(b []byte) { data = append(data, b...) },
	}

	cmd := "/usr/bin/osascript"
	args := []string{"-l", "JavaScript", "-e", script}

	if len(argv) > 0 {
		args = append(args, argv...)
	}

	if err := d.Run(exec.Command(cmd, args...)); err != nil {
		return data, err
	}

	return data, nil
}

// runJXA2JSON executes a JXA script and unmarshals its output to target using
// json.Unmarshal()
func runJXA2JSON(script string, target interface{}, argv ...string) error {
	data, err := runJXA(script, argv...)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, target); err != nil {
		return err
	}

	return nil
}
