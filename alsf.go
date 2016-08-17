//
// Copyright (c) 2016 Dean Jackson <deanishe@deanishe.net>
//
// MIT Licence. See http://opensource.org/licenses/MIT
//
// Created on 2016-05-30
//

// TODO: Fix maxResults: it currently only takes effect if there's a query
// TODO: Open Bookmark in Private Mode
// TODO: Duplicate tab
// TODO: Duplicate tab in Private Mode
// TODO: Open URL in Chrome etc.
// TODO: iCloud tabs (~/Library/SyncedPreferences/com.apple.Safari.plist)

package main // import "gogs.deanishe.net/deanishe/alfred-safari"

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/juju/deputy"
	"gogs.deanishe.net/deanishe/awgo"
	"gogs.deanishe.net/deanishe/awgo/fuzzy"
	"gogs.deanishe.net/deanishe/go-safari"
	"gopkg.in/alecthomas/kingpin.v2"
)

// Defaults for Kingpin flags
const (
	defaultMaxResults  = "100"
	defaultMinScore    = "30"
	defaultMaxCacheAge = "5s"
)

var (
	// Kingpin and script options
	app *kingpin.Application

	// Application commands
	actvCmd, bkCmd, brwsCmd, clsCmd       *kingpin.CmdClause
	fldrsCmd, openCmd, rlCmd, tbCmd       *kingpin.CmdClause
	distCmd, actCmd, tabActCmd, urlActCmd *kingpin.CmdClause
	lstCmd, lstTabActCmd, lstURLActCmd    *kingpin.CmdClause
	currCmd                               *kingpin.CmdClause

	// Script options (populated by Kingpin application)
	query        string
	left, right  bool
	window, tab  int
	action, uid  string
	actionURL    *url.URL
	minimumScore float64
	maxCacheAge  time.Duration
	maxResults   int

	// Workflow stuff
	wf           *workflow.Workflow
	tabCachePath string
	scriptDirs   []string

	urlKillWords = []string{"www.", ".com", ".net", ".org", ".co.uk"}
)

// Mostly sets up kingpin commands
func init() {

	safari.DefaultOptions.IgnoreBookmarklets = true

	wf = workflow.NewWorkflow(nil)
	tabCachePath = filepath.Join(wf.CacheDir(), "tabs.json")
	scriptDirs = []string{
		filepath.Join(wf.Dir(), "scripts", "tab"),
		filepath.Join(wf.Dir(), "scripts", "url"),
		filepath.Join(wf.DataDir(), "scripts", "tab"),
		filepath.Join(wf.DataDir(), "scripts", "url"),
	}

	app = kingpin.New("alsf", "Safari bookmarks, windows and tabs in Alfred.")
	app.HelpFlag.Short('h')
	app.Version(wf.Version())

	// ---------------------------------------------------------------
	// List action commands
	lstCmd = app.Command("actions", "List actions.").Alias("la")
	lstTabActCmd = lstCmd.Command("tab", "List tab actions.").Alias("lta")
	lstURLActCmd = lstCmd.Command("url", "List URL actions.").Alias("lua")

	// ---------------------------------------------------------------
	// Action commands
	actCmd = app.Command("action", "Run an action.").Alias("A")
	tabActCmd = actCmd.Command("tab", "Run a tab action.").Alias("t")
	urlActCmd = actCmd.Command("url", "Run a URL action.").Alias("u")
	// Common URL options
	for _, cmd := range []*kingpin.CmdClause{urlActCmd, lstURLActCmd} {
		cmd.Flag("url", "URL to action.").Short('u').Required().URLVar(&actionURL)
	}
	// Common action options
	for _, cmd := range []*kingpin.CmdClause{tabActCmd, urlActCmd} {
		cmd.Flag("action", "Action name.").Short('a').PlaceHolder("NAME").Required().StringVar(&action)
	}

	// ---------------------------------------------------------------
	// Commands using window and tab
	actvCmd = app.Command("activate", "Active a specific window or tab.").Alias("a")
	clsCmd = app.Command("close", "Close tab(s).").Alias("c")

	// Common options
	for _, cmd := range []*kingpin.CmdClause{actvCmd, clsCmd, tabActCmd, lstTabActCmd} {
		cmd.Flag("window", "Window number.").
			Short('w').Default("1").IntVar(&window)
		cmd.Flag("tab", "Tab number.").
			Short('t').Required().IntVar(&tab)
	}
	clsCmd.Flag("left", "Close tab(s) to left of specified tab.").
		Short('l').BoolVar(&left)
	clsCmd.Flag("right", "Close tab(s) to right of specified tab.").
		Short('r').BoolVar(&right)

	// ---------------------------------------------------------------
	// Commands using UID
	brwsCmd = app.Command("browse", "Filter the contents of a bookmark folder.").Alias("B")
	openCmd = app.Command("open", "Open bookmark(s) or folder(s).").Alias("o")
	// Common options
	for _, cmd := range []*kingpin.CmdClause{brwsCmd, openCmd} {
		cmd.Flag("uid", "Bookmark/folder UID.").Short('u').StringVar(&uid)
	}

	// ---------------------------------------------------------------
	// Commands using query etc.
	bkCmd = app.Command("bookmarks", "Filter your bookmarks.").Alias("b")
	fldrsCmd = app.Command("folders", "Filter your bookmark folders.").Alias("f")
	rlCmd = app.Command("reading-list", "Filter your Reading List.").Alias("r")
	tbCmd = app.Command("tabs", "Filter your tabs.").Alias("t")
	// Common options
	for _, cmd := range []*kingpin.CmdClause{bkCmd, brwsCmd, fldrsCmd, rlCmd, tbCmd, lstTabActCmd, lstURLActCmd} {
		cmd.Flag("query", "Search query.").Short('q').StringVar(&query)
		cmd.Flag("max-results", "Maximum number of results to send to Alfred.").
			Short('r').Default(defaultMaxResults).IntVar(&maxResults)
		cmd.Flag("min-score", "Minimum score for search matches.").
			Short('s').Default(defaultMinScore).Float64Var(&minimumScore)
	}
	tbCmd.Flag("max-cache", "Maximum time to cache tab list for.").
		Short('c').Default(defaultMaxCacheAge).DurationVar(&maxCacheAge)

	// ---------------------------------------------------------------
	// Other commands
	currCmd = app.Command("active-tab", "Generate workflow variables for active tab.").Alias("at")
	distCmd = app.Command("distname", "Print name for .alfredworkflow file.").Alias("dn")

	app.DefaultEnvars()
}

// urlKeywords returns fuzzy keywords for URL.
func urlKeywords(URL string) string {
	u, err := url.Parse(URL)
	if err != nil {
		return ""
	}
	h := u.Host
	for _, s := range urlKillWords {
		h = strings.Replace(h, s, "", -1)
	}
	return h
}

// fuzzyActions makes Action fuzzy sortable.
type fuzzyActions []Actionable

// fuzzy.Interface
func (a fuzzyActions) Len() int              { return len(a) }
func (a fuzzyActions) Less(i, j int) bool    { return a[i].Title() < a[j].Title() }
func (a fuzzyActions) Swap(i, j int)         { a[i], a[j] = a[j], a[i] }
func (a fuzzyActions) Keywords(i int) string { return a[i].Title() }

// fuzzyBookmarks makes safari.Bookmark fuzzy sortable.
type fuzzyBookmarks []*safari.Bookmark

// fuzzy.Interface
func (b fuzzyBookmarks) Len() int           { return len(b) }
func (b fuzzyBookmarks) Less(i, j int) bool { return b[i].Title < b[j].Title }
func (b fuzzyBookmarks) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b fuzzyBookmarks) Keywords(i int) string {
	return fmt.Sprintf("%s %s", b[i].Title, urlKeywords(b[i].URL))
}

// fuzzyFolders makes safari.Folder fuzzy sortable.
type fuzzyFolders []*safari.Folder

// fuzzy.Interface
func (f fuzzyFolders) Len() int              { return len(f) }
func (f fuzzyFolders) Less(i, j int) bool    { return f[i].Title < f[j].Title }
func (f fuzzyFolders) Swap(i, j int)         { f[i], f[j] = f[j], f[i] }
func (f fuzzyFolders) Keywords(i int) string { return f[i].Title }

// fuzzyTabs makes safari.Tab fuzzy sortable.
type fuzzyTabs []*safari.Tab

// fuzzy.Interface
func (t fuzzyTabs) Len() int           { return len(t) }
func (t fuzzyTabs) Less(i, j int) bool { return t[i].Title < t[j].Title }
func (t fuzzyTabs) Swap(i, j int)      { t[i], t[j] = t[j], t[i] }
func (t fuzzyTabs) Keywords(i int) string {
	kw := fmt.Sprintf("%s %s", t[i].Title, urlKeywords(t[i].URL))
	log.Printf("kw=%v", kw)
	return kw
}

// fuzzyProxy allows Bookmarks and Folders to be sorted together
type fuzzyProxy struct {
	uid      string
	keywords string
}

// fuzzyProxies allows Bookmarks and Folders to be sorted together
type fuzzyProxies []*fuzzyProxy

// fuzzy.Interface
func (f fuzzyProxies) Len() int              { return len(f) }
func (f fuzzyProxies) Less(i, j int) bool    { return f[i].keywords < f[j].keywords }
func (f fuzzyProxies) Swap(i, j int)         { f[i], f[j] = f[j], f[i] }
func (f fuzzyProxies) Keywords(i int) string { return f[i].keywords }

// loadWindows returns a list of Safari windows and caches the results for a few seconds.
func loadWindows() ([]*safari.Window, error) {

	// Return cached data if they exist and are fresh enough
	if fi, err := os.Stat(tabCachePath); err == nil {

		age := time.Since(fi.ModTime())
		if age <= maxCacheAge {

			data, err := ioutil.ReadFile(tabCachePath)
			if err != nil {
				return nil, err
			}

			wins := []*safari.Window{}
			if err := json.Unmarshal(data, &wins); err != nil {
				return nil, err
			}

			log.Printf("Loaded tab list from cache (%v old)", age)

			return wins, nil
		}

		log.Printf("Cache expired (%v > %v)", age, maxCacheAge)
	}

	wins, err := safari.Windows()
	if err != nil {
		return nil, err
	}

	// Cache data
	data, err := json.MarshalIndent(wins, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := ioutil.WriteFile(tabCachePath, data, 0600); err != nil {
		return nil, err
	}
	log.Printf("Saved tab list: %s", workflow.ShortenPath(tabCachePath))

	return wins, nil
}

// invalidateCache deletes the cached tab list. Call when opening/closing tabs.
func invalidateCache() {
	if err := os.Remove(tabCachePath); err != nil {
		log.Printf("Error removing cached tab list: %v", err)
	}
}

// openURL opens URL in user's default browser.
func openURL(URL string) error {

	d := deputy.Deputy{
		Errors:    deputy.FromStderr,
		StdoutLog: func(b []byte) {},
	}

	cmd := "open"
	args := []string{URL}

	if err := d.Run(exec.Command(cmd, args...)); err != nil {
		return err
	}

	return nil
}

// removeBookmarklets returns slice bookmarks with any bookmarklets removed.
// Bookmarklets are any Bookmarks whose URLs start with "javascript:".
//
// TODO: Run bookmarklets in the current browser tab
// func removeBookmarklets(bookmarks []*safari.Bookmark) []*safari.Bookmark {
// 	r := []*safari.Bookmark{}
// 	i := 0
// 	for _, bm := range bookmarks {
// 		if !strings.HasPrefix(bm.URL, "javascript:") {
// 			r = append(r, bm)
// 		} else {
// 			i++
// 		}
// 	}
//
// 	return r
// }

// doActivate activates the specified window (and tab).
func doActivate() error {

	log.Printf("Activating %dx%d", window, tab)

	return safari.ActivateTab(window, tab)
}

// doOpen opens the bookmark(s)/folder(s) with the specified UIDs.
func doOpen() error {

	if uid == "" {
		log.Println("No UID specified")
		return nil
	}

	invalidateCache()

	log.Printf("Searching for %v ...", uid)

	if bm := safari.BookmarkForUID(uid); bm != nil {
		log.Printf("Opening \"%s\" (%s) ...", bm.Title, bm.URL)
		return openURL(bm.URL)
	}

	if f := safari.FolderForUID(uid); f != nil {

		errs := []error{}

		for _, bm := range f.Bookmarks {
			log.Printf("Opening \"%s\" (%s) ...", bm.Title, bm.URL)
			if err := openURL(bm.URL); err != nil {
				log.Printf("Error opening bookmark: %v", err)
				errs = append(errs, err)
			}
		}

		if len(errs) > 0 {
			return errs[0]
		}

		return nil
	}

	return fmt.Errorf("Not found: %s", uid)
}

// doTabs filters tabs and outputs Alfred results.
func doTabs() error {

	log.Printf("query=%s", query)

	wins, err := loadWindows()
	if err != nil {
		return err
	}

	tabs := fuzzyTabs{}
	for _, w := range wins {
		for _, t := range w.Tabs {
			tabs = append(tabs, t)
		}
	}
	log.Printf("%d tabs", len(tabs))

	if query != "" {
		scores := fuzzy.Sort(tabs, query)
		for i, s := range scores {
			if s < minimumScore {
				tabs = tabs[:i]
				break
			}
			log.Printf("[%0.1f] %s", s, tabs[i].Title)

		}
		log.Printf("%d tabs match `%s`", len(tabs), query)
	}

	if len(tabs) == 0 {
		wf.Warn("No tabs found", "Try a different query?")
		return nil
	}

	for _, t := range tabs {

		it := wf.NewItem(t.Title)
		it.Subtitle = t.URL
		it.Valid = true

		if t.Active {
			it.SetIcon("active.png", "")
		} else {
			it.SetIcon("icon.png", "")
		}

		it.SetVar("ALSF_WINDOW", fmt.Sprintf("%d", t.WindowIndex))
		it.SetVar("ALSF_TAB", fmt.Sprintf("%d", t.Index))
		it.SetVar("action", "activate")

		m := it.NewModifier("cmd")
		m.SetSubtitle("Other actions…")
		m.SetVar("ALSF_INDEX", fmt.Sprintf("%02dx%02d", t.WindowIndex, t.Index))
		m.SetVar("action", "actions")

		// m := it.NewModifier("cmd")
		// m.SetSubtitle("Close tab")
		// m.SetVar("action", "close")

		// m = it.NewModifier("shift")
		// m.SetSubtitle("Close other tab(s)")
		// m.SetVar("ALSF_RIGHT", "1")
		// m.SetVar("ALSF_LEFT", "1")
		// m.SetVar("action", "close")

		// m = it.NewModifier("ctrl")
		// m.SetSubtitle("Close tab(s) to the left")
		// m.SetVar("ALSF_LEFT", "1")
		// m.SetVar("action", "close")

		// m = it.NewModifier("alt")
		// m.SetSubtitle("Close tab(s) to the right")
		// m.SetVar("ALSF_RIGHT", "1")
		// m.SetVar("action", "close")

	}
	wf.SendFeedback()

	return nil
}

// bookmarkItem returns a feedback Item for Safari Bookmark.
func bookmarkItem(bm *safari.Bookmark) *workflow.Item {

	it := wf.NewItem(bm.Title)

	it.Subtitle = bm.URL
	it.Arg = bm.URL
	it.UID = bm.UID
	it.Valid = true

	it.Copytext = bm.URL
	it.Largetext = bm.Preview

	if bm.InReadingList() {
		it.SetIcon("reading-list.png", "")
	} else {
		it.SetIcon("com.apple.safari.bookmark", "filetype")
	}

	it.SetVar("ALSF_UID", bm.UID)
	it.SetVar("action", "open")

	return it
}

// folderSubtitle generates a subtitle for a Folder.
func folderSubtitle(f *safari.Folder) string {
	s := []string{}
	for _, f2 := range f.Ancestors {
		s = append(s, f2.Title)
	}
	return strings.Join(s, " / ")
}

// folderTitle generates a title for a Folder.
func folderTitle(f *safari.Folder) string {
	return fmt.Sprintf("%s (%d bookmarks)", f.Title, len(f.Bookmarks))
}

// folderItem returns a feedback Item for Safari Folder.
func folderItem(f *safari.Folder) *workflow.Item {

	it := wf.NewItem(folderTitle(f))
	it.Subtitle = folderSubtitle(f)
	it.SetIcon("public.folder", "filetype")

	// Make folder actionable if it isn't empty
	if len(f.Bookmarks)+len(f.Folders) > 0 {
		it.Valid = true
		it.SetVar("ALSF_UID", f.UID)

		// Allow opening folder if it contains bookmarks
		m := it.NewModifier("cmd")

		if len(f.Bookmarks) > 0 {

			m.SetSubtitle(fmt.Sprintf("Open %d bookmark(s)", len(f.Bookmarks)))
			// m.SetVar("open_bookmark", "1")
			m.SetVar("action", "open")

		} else {
			m.SetValid(false)
		}
		// Default only
		// it.SetVar("browse_folder", "1")
		it.SetVar("action", "browse")
	}

	return it
}

// doFolders filters bookmark folders and outputs Alfred results.
func doFolders() error {

	log.Printf("query=%s", query)

	sf := safari.Folders()
	ff := make(fuzzyFolders, len(sf))

	for i, f := range sf {
		ff[i] = f
	}

	if query != "" {
		scores := fuzzy.Sort(ff, query)
		for i, s := range scores {
			if s < minimumScore {
				ff = ff[:i]
				break
			}
			if i == maxResults {
				log.Printf("Reached max. results (%d)", maxResults)
				ff = ff[:i]
				break
			}
		}
		log.Printf("%d folders match \"%s\"", len(ff), query)
	}

	if len(ff) == 0 {
		wf.Warn("No folders found", "Try a different query?")
		return nil
	}

	// Send results
	log.Printf("Sending %d results to Alfred ...", len(ff))
	for _, f := range ff {
		folderItem(f)
	}

	wf.SendFeedback()
	return nil
}

// doBrowse filters the contents of a folder and outputs Alfred results.
func doBrowse() error {

	log.Printf("query=%s", query)

	// ----------------------------------------------------------------
	// Gather results

	f := safari.FolderForUID(uid)
	if f == nil {
		return fmt.Errorf("No folder found with UID: %s", uid)
	}

	log.Printf("%d folders, %d bookmarks in \"%s\"", len(f.Folders), len(f.Bookmarks), f.Title)

	// ----------------------------------------------------------------
	// Show "Back" options if query is empty
	if query == "" {

		if len(f.Ancestors) > 0 {

			p := f.Ancestors[len(f.Ancestors)-1]

			it := wf.NewItem(fmt.Sprintf("Up to \"%s\"", p.Title))
			it.SetIcon("up.png", "")
			it.Valid = true
			it.SetVar("ALSF_UID", p.UID)

			// Alternate action: Go to All Folders
			m := it.NewModifier("cmd")
			m.SetSubtitle("Go back to All Folders")
			// m.SetVar("back_to_root", "1")
			m.SetVar("action", "top")

			// Default only
			it.SetVar("action", "browse")
			// it.SetVar("browse_folder", "1")
		} else if uid != "" { // One of the top-level items, e.g. Favorites
			it := wf.NewItem("Back to All Folders")
			it.Valid = true
			it.SetIcon("home.png", "")
			// it.SetVar("back_to_root", "1")
			it.SetVar("action", "top")
		}
	}

	// ----------------------------------------------------------------
	// Sort Folders and Bookmarks

	proxies := fuzzyProxies{}
	fMap := map[string]*safari.Folder{}
	bMap := map[string]*safari.Bookmark{}

	for _, f2 := range f.Folders {
		fMap[f2.UID] = f2
		proxies = append(proxies, &fuzzyProxy{f2.UID, f2.Title})
	}
	for _, bm := range f.Bookmarks {
		bMap[bm.UID] = bm
		proxies = append(proxies, &fuzzyProxy{bm.UID, bm.Title})
	}

	if query != "" { // Do fuzzy sort
		scores := fuzzy.Sort(proxies, query)
		for i, s := range scores {
			if s < minimumScore {
				proxies = proxies[:i]
				break
			}
			log.Printf("[%0.1f] %s", s, proxies[i].keywords)

			if i == maxResults {
				log.Printf("Reached max. results (%d)", maxResults)
				proxies = proxies[:i]
				break
			}
		}
	} else { // Sort by title anyway
		sort.Sort(proxies)
	}

	// ----------------------------------------------------------------
	// Output results

	if len(proxies) == 0 {
		wf.Warn("No bookmarks or folders found", "Try a different query?")
		return nil
	}

	log.Printf("Sending %d results to Alfred...", len(proxies))

	for _, p := range proxies {

		if f, ok := fMap[p.uid]; ok {
			folderItem(f)
		} else {
			bookmarkItem(bMap[p.uid])
		}
	}

	wf.SendFeedback()

	return nil
}

// filterBookmarks filters bookmarks and outputs Alfred results.
func filterBookmarks(bookmarks []*safari.Bookmark) error {

	log.Printf("query=%s", query)

	log.Printf("Loaded %d bookmarks", len(bookmarks))

	bms := make(fuzzyBookmarks, len(bookmarks))
	for i, bm := range bookmarks {
		bms[i] = bm
	}

	if query != "" {
		// log.Printf("Searching %d bookmarks for \"%s\" ...", len(bms), query)
		scores := fuzzy.Sort(bms, query)
		for i, s := range scores {
			if s < minimumScore {
				bms = bms[:i]
				break
			}
			log.Printf("[%0.1f] %s", s, bms[i].Title)

			if i == maxResults {
				log.Printf("Reached max. results (%d)", maxResults)
				bms = bms[:i]
				break
			}
		}

		log.Printf("%d bookmark(s) matching \"%s\"", len(bms), query)

	}

	if len(bms) == 0 {
		wf.Warn("No bookmarks found", "Try a different query?")
		return nil
	}

	// Display results
	log.Printf("Sending %d results to Alfred ...", len(bms))
	for _, bm := range bms {
		bookmarkItem(bm)
	}

	wf.SendFeedback()
	return nil
}

// doBookmarks filters bookmarks and outputs Alfred results.
func doBookmarks() error { return filterBookmarks(safari.Bookmarks()) }

// doReadingList filters Safari's Reading List and sends results to Alfred.
func doReadingList() error { return filterBookmarks(safari.ReadingList().Bookmarks) }

// doClose closes the specified tab(s).
// TODO: Activate tab after closing to left or right?
func doClose() error {

	invalidateCache()

	if !left && !right { // Close current tab
		log.Printf("Closing tab %d of window %d ...", tab, window)
		return safari.CloseTab(window, tab)
	}

	if left && right { // Close all other tabs
		log.Printf("Closing all tabs in window %d except %d ...", window, tab)
		return safari.CloseTabsOther(window, tab)
	}

	if left {
		log.Printf("Closing all tabs in window %d to left of %d ...", window, tab)
		return safari.CloseTabsLeft(window, tab)
	}

	if right {
		log.Printf("Closing all tabs in window %d to right of %d ...", window, tab)
		return safari.CloseTabsRight(window, tab)
	}

	return nil
}

// doURLAction performs an action on a URL.
func doURLAction() error {
	log.Printf("URL=%s, action=%s", actionURL, action)

	LoadScripts(scriptDirs...)

	a := URLAction(action)
	if a == nil {
		return fmt.Errorf("Unknown action : %s", action)
	}
	return a.Run(actionURL)
}

// doTabAction performs an action on a tab.
func doTabAction() error {
	log.Printf("window=%d, tab=%d, action=%s", window, tab, action)

	LoadScripts(scriptDirs...)

	a := TabAction(action)
	if a == nil {
		return fmt.Errorf("Unknown action : %s", action)
	}
	wins, err := loadWindows()
	if err != nil {
		return err
	}
	for _, w := range wins {
		if w.Index == window {
			for _, t := range w.Tabs {
				if t.Index == tab {
					return a.Run(t)
				}
			}
		}
	}
	return fmt.Errorf("Tab not found : %02dx%02d", window, tab)
}

// listActions sends a list of actions to Alfred.
func listActions(actions []Actionable) error {
	log.Printf("query=%s", query)

	acts := make(fuzzyActions, len(actions))
	for i, a := range actions {
		acts[i] = a
	}

	if query != "" {
		scores := fuzzy.Sort(acts, query)
		for i, s := range scores {
			if s < minimumScore {
				acts = acts[:i]
				break
			}
			log.Printf("[%0.1f] %s", s, acts[i].Title())

			if i == maxResults {
				log.Printf("Reached max. results (%d)", maxResults)
				acts = acts[:i]
				break
			}
		}

		log.Printf("%d action(s) matching \"%s\"", len(acts), query)

	}

	if len(acts) == 0 {
		wf.Warn("No actions found", "Try a different query?")
		return nil
	}

	for _, a := range acts {

		it := wf.NewItem(a.Title())
		it.Arg = a.Title()
		it.SetIcon(a.Icon(), a.IconType())
		it.Valid = true
		it.SetVar("ALSF_ACTION", a.Title())

		if _, ok := a.(TabActionable); ok {
			it.SetVar("ACTION_TYPE", "tab")
			it.SetVar("ALSF_WINDOW", fmt.Sprintf("%d", window))
			it.SetVar("ALSF_TAB", fmt.Sprintf("%d", tab))
		} else {
			it.SetVar("ACTION_TYPE", "url")
			it.SetVar("ALSF_URL", actionURL.String())
		}

	}
	wf.SendFeedback()
	return nil
}

func doListURLActions() error {
	LoadScripts(scriptDirs...)

	acts := make([]Actionable, len(URLActions()))
	for i, a := range URLActions() {
		acts[i] = a
	}
	return listActions(acts)
}

func doListTabActions() error {
	LoadScripts(scriptDirs...)

	acts := make([]Actionable, len(TabActions()))
	for i, a := range TabActions() {
		acts[i] = a
	}
	return listActions(acts)
}

// doCurrentTab outputs workflow variables for the current tab.
func doCurrentTab() error {
	wins, err := loadWindows()
	if err != nil {
		return err
	}
	if len(wins) == 0 {
		return fmt.Errorf("No windows.")
	}

	vs := &workflow.VarSet{}
	vs.Var("ALSF_WINDOW", "1")
	// Find active tab
	for _, w := range wins {
		if w.Index == 1 {
			vs.Var("ALSF_TAB", fmt.Sprintf("%d", w.ActiveTab))
			break
		}
	}
	s, err := vs.String()
	if err != nil {
		return err
	}
	_, err = fmt.Println(s)
	return err
}

// doDistname prints the filename of the .alfredworkflow file to STDOUT.
func doDistname() error {
	fmt.Println(strings.Replace(
		fmt.Sprintf("%s %s.alfredworkflow", wf.Name(), wf.Version()),
		" ", "-", -1))
	return nil
}

// run is the main script entry point. It's called from main.
func run() {
	var err error

	cmd, err := app.Parse(os.Args[1:])
	if err != nil {
		wf.FatalError(err)
	}

	switch cmd {

	case actvCmd.FullCommand():
		err = doActivate()

	case bkCmd.FullCommand():
		err = doBookmarks()

	case brwsCmd.FullCommand():
		err = doBrowse()

	case clsCmd.FullCommand():
		err = doClose()

	case fldrsCmd.FullCommand():
		err = doFolders()

	case openCmd.FullCommand():
		err = doOpen()

	case rlCmd.FullCommand():
		err = doReadingList()

	case tbCmd.FullCommand():
		err = doTabs()

	case distCmd.FullCommand():
		err = doDistname()

	case urlActCmd.FullCommand():
		err = doURLAction()

	case tabActCmd.FullCommand():
		err = doTabAction()

	case lstTabActCmd.FullCommand():
		err = doListTabActions()

	case lstURLActCmd.FullCommand():
		err = doListURLActions()
	case currCmd.FullCommand():
		err = doCurrentTab()

	default:
		err = fmt.Errorf("Unknown command: %s", cmd)

	}

	if err != nil {
		wf.FatalError(err)
	}
}

// main wraps run() (the actual entry point) to catch errors.
func main() {
	wf.Run(run)
}
