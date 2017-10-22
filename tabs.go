//
// Copyright (c) 2017 Dean Jackson <deanishe@deanishe.net>
//
// MIT Licence. See http://opensource.org/licenses/MIT
//
// Created on 2017-10-22
//

package main

import (
	"fmt"
	"log"
	"net/url"
	"strings"

	safari "git.deanishe.net/deanishe/go-safari"
	aw "github.com/deanishe/awgo"
)

// Activate the specified window (and tab).
func doActivate() error {

	wf.TextErrors = true

	log.Printf("Activating %dx%d", winIdx, tabIdx)

	return safari.ActivateTab(winIdx, tabIdx)
}

// doFilterTabActions is a Script Filter for tab actions.
func doFilterTabActions() error {

	log.Printf("url=%s, scheme=%s", actionURL, actionURL.Scheme)

	LoadScripts(scriptDirs...)
	acts := []Actionable{}
	for _, a := range TabActions() {
		acts = append(acts, a)
	}

	// No URL actions for favorites:// and bookmarks:// etc.
	if actionURL.Scheme == "http" || actionURL.Scheme == "https" {
		for _, a := range URLActions() {
			acts = append(acts, a)
		}
	}
	return listActions(acts)
}

// doTabAction performs an action on a tab.
func doTabAction() error {
	wf.TextErrors = true

	var (
		err error
		tab *safari.Tab
		URL *url.URL
	)

	log.Printf("window=%d, tab=%d, action=%s", winIdx, tabIdx, action)

	wins, err := loadWindows()
	if err != nil {
		return err
	}

	for _, w := range wins {
		if tab != nil {
			break
		}
		if w.Index == winIdx {
			for _, t := range w.Tabs {
				if t.Index == tabIdx {
					tab = t
					URL, err = url.Parse(t.URL)
					if err != nil {
						return err
					}
					break
				}
			}
		}
	}
	if tab == nil {
		return fmt.Errorf("Tab not found : %02dx%02d", winIdx, tabIdx)
	}

	if actionType == "bookmarklet" {
		bm := safari.BookmarkForUID(action)
		if bm == nil {
			return fmt.Errorf("Unknown bookmarklet: %s", action)
		}
		js, err := bm.ToJS()
		if err != nil {
			return err
		}
		return tab.RunJS(js)
	}

	LoadScripts(scriptDirs...)
	if actionType == "tab" {
		ta := TabAction(action)
		if ta == nil {
			return fmt.Errorf("Unknown action : %s", action)
		}
		return ta.Run(tab)
	}

	if actionType == "url" {
		ua := URLAction(action)
		if ua == nil {
			return fmt.Errorf("Unknown action : %s", action)
		}
		return ua.Run(URL)
	}

	return fmt.Errorf("Tab not found : %02dx%02d", winIdx, tabIdx)
}

// doCurrentTab outputs workflow variables for the current tab.
func doCurrentTab() error {
	tab, err := safari.ActiveTab()
	if err != nil {
		return fmt.Errorf("Couldn't get active tab: %s", err)
	}
	log.Printf("%v", tab)
	av := aw.NewArgVars()
	av.Var("ALSF_WINDOW", "1").
		Var("ALSF_TAB", fmt.Sprintf("%d", tab.Index)).
		Var("ALSF_URL", tab.URL)

	s, err := av.String()
	if err != nil {
		return err
	}

	_, err = fmt.Println(s)
	return err
}

// Filter tabs and output Alfred results.
func doFilterTabs() error {

	log.Printf("query=%s", query)

	wins, err := loadWindows()
	if err != nil {
		return err
	}

	for _, w := range wins {
		for _, t := range w.Tabs {

			it := wf.NewItem(t.Title).
				Subtitle(t.URL).
				Valid(true).
				Match(fmt.Sprintf("%s %s", t.Title, urlKeywords(t.URL)))

			if t.Active {
				it.Icon(IconActive)
			} else {
				it.Icon(IconTab)
			}

			it.Var("ALSF_WINDOW", fmt.Sprintf("%d", t.WindowIndex)).
				Var("ALSF_TAB", fmt.Sprintf("%d", t.Index)).
				Var("ALSF_URL", t.URL).
				Var("action", "activate")

			it.NewModifier("cmd").
				Subtitle("Other actions…").
				Var("action", "tab-actions")

			it = customTabActions(it)
		}
	}

	if query != "" {
		res := wf.Filter(query)
		log.Printf("%d results for `%s`", len(res), query)
	}
	wf.WarnEmpty("No tabs found", "Try a different query?")
	wf.SendFeedback()

	return nil
}

// doClose closes the specified tab(s).
// TODO: Activate tab after closing to left or right?
func doClose() error {

	if !left && !right { // Close current tab
		log.Printf("Closing tab %d of window %d ...", tabIdx, winIdx)
		return safari.CloseTab(winIdx, tabIdx)
	}

	if left && right { // Close all other tabs
		log.Printf("Closing all tabs in window %d except %d ...", winIdx, tabIdx)
		return safari.CloseTabsOther(winIdx, tabIdx)
	}

	if left {
		log.Printf("Closing all tabs in window %d to left of %d ...", winIdx, tabIdx)
		return safari.CloseTabsLeft(winIdx, tabIdx)
	}

	if right {
		log.Printf("Closing all tabs in window %d to right of %d ...", winIdx, tabIdx)
		return safari.CloseTabsRight(winIdx, tabIdx)
	}

	return nil
}

// --------------------------------------------------------------------
// Helpers

// customTabActions adds user-specified actions/bookmarklets to tab Item.
func customTabActions(it *aw.Item) *aw.Item {

	var (
		// UID:title map
		bkms = map[string]string{}
		// Name:Type map
		actions = map[string]string{}
	)
	for _, bm := range safari.FilterBookmarks(func(bm *safari.Bookmark) bool { return bm.IsBookmarklet() }) {
		bkms[bm.UID()] = bm.Title()
	}

	LoadScripts(scriptDirs...)
	for _, a := range TabActions() {
		actions[a.Title()] = "tab"
	}
	for _, a := range URLActions() {
		actions[a.Title()] = "url"
	}

	actionDetails := func(s string) (title, typ, action string) {
		if strings.HasPrefix(s, "bkm:") {
			action = s[4:]
			title = bkms[action]
			typ = "bookmarklet"
			return
		}
		title = s
		typ = actions[title]
		action = title
		return
	}

	altActions := []struct {
		action, key string
	}{
		{tabActionCtrl, "ctrl"},
		{tabActionShift, "shift"},
		{tabActionOpt, "alt"},
		{tabActionFn, "fn"},
	}

	for _, a := range altActions {
		if a.action == "" { // unset
			continue
		}
		title, typ, action := actionDetails(a.action)
		if typ == "" {
			log.Printf("Unknown action %s", a.action)
			continue
		}
		it.NewModifier(a.key).
			Subtitle(title).
			Valid(true).
			Var("action", "tab-action").
			Var("ALSF_ACTION", action).
			Var("ALSF_ACTION_TYPE", typ)
	}

	return it
}

// runBookmarklet executes a bookmarklet in the current tab.
func runBookmarklet(bm *safari.Bookmark) error {
	tab, err := safari.ActiveTab()
	if err != nil {
		return err
	}
	js, err := bm.ToJS()
	if err != nil {
		return err
	}
	return tab.RunJS(js)
}
