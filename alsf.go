//
// Copyright (c) 2016 Dean Jackson <deanishe@deanishe.net>
//
// MIT Licence. See http://opensource.org/licenses/MIT
//
// Created on 2016-05-30
//

// TODO: Other Actions… for URLs (bookmarks)
// TODO: Allow user to configure URL/tab actions for other modifiers
// TODO: Bookmarklets
// TODO: Script: Open Bookmark/URL in Private Mode
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
	"strings"
	"time"

	"github.com/juju/deputy"
	"gogs.deanishe.net/deanishe/awgo"
	"gogs.deanishe.net/deanishe/go-safari"
	"gopkg.in/alecthomas/kingpin.v2"
)

// Defaults for Kingpin flags
const (
	defaultMaxResults  = "100"
	defaultMinScore    = "30"
	defaultMaxCacheAge = "5s"
)

// Icons
var (
	IconActive      = &aw.Icon{Value: "active.png"}
	IconDefault     = &aw.Icon{Value: "icon.png"}
	IconReadingList = &aw.Icon{Value: "reading-list.png"}
	IconBookmark    = &aw.Icon{Value: "com.apple.safari.bookmark", Type: "filetype"}
	IconFolder      = &aw.Icon{Value: "public.folder", Type: "filetype"}
	IconUp          = &aw.Icon{Value: "up.png"}
	IconHome        = &aw.Icon{Value: "home.png"}
)

var (
	// Kingpin and script options
	app *kingpin.Application

	// Application commands
	activateCmd, filterBookmarksCmd, filterFolderCmd, closeCmd        *kingpin.CmdClause
	filterAllFoldersCmd, openCmd, filterReadingListCmd, filterTabsCmd *kingpin.CmdClause
	distnameCmd, runActionCmd, runTabActionCmd, runURLActionCmd       *kingpin.CmdClause
	filterActionsCmd, filterTabActionsCmd, filterURLActionsCmd        *kingpin.CmdClause
	activeTabCmd                                                      *kingpin.CmdClause

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
	wf           *aw.Workflow
	tabCachePath string
	scriptDirs   []string

	urlKillWords = []string{"www.", ".com", ".net", ".org", ".co.uk"}
)

// Mostly sets up kingpin commands
func init() {

	safari.DefaultOptions.IgnoreBookmarklets = true

	wf = aw.NewWorkflow(nil)
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
	filterActionsCmd = app.Command("actions", "List actions.").Alias("la")
	filterTabActionsCmd = filterActionsCmd.Command("tab", "List tab actions.").Alias("lta")
	filterURLActionsCmd = filterActionsCmd.Command("url", "List URL actions.").Alias("lua")

	// ---------------------------------------------------------------
	// Action commands
	runActionCmd = app.Command("action", "Run an action.").Alias("A")
	runTabActionCmd = runActionCmd.Command("tab", "Run a tab action.").Alias("t")
	runURLActionCmd = runActionCmd.Command("url", "Run a URL action.").Alias("u")
	// Common URL options
	for _, cmd := range []*kingpin.CmdClause{runURLActionCmd, filterURLActionsCmd} {
		cmd.Flag("url", "URL to action.").Short('u').Required().URLVar(&actionURL)
	}
	// Common action options
	for _, cmd := range []*kingpin.CmdClause{runTabActionCmd, runURLActionCmd} {
		cmd.Flag("action", "Action name.").Short('a').PlaceHolder("NAME").Required().StringVar(&action)
	}

	// ---------------------------------------------------------------
	// Commands using window and tab
	activateCmd = app.Command("activate", "Active a specific window or tab.").Alias("a")
	closeCmd = app.Command("close", "Close tab(s).").Alias("c")

	// Common options
	for _, cmd := range []*kingpin.CmdClause{activateCmd, closeCmd, runTabActionCmd, filterTabActionsCmd} {
		cmd.Flag("window", "Window number.").
			Short('w').Default("1").IntVar(&window)
		cmd.Flag("tab", "Tab number.").
			Short('t').Required().IntVar(&tab)
	}
	closeCmd.Flag("left", "Close tab(s) to left of specified tab.").
		Short('l').BoolVar(&left)
	closeCmd.Flag("right", "Close tab(s) to right of specified tab.").
		Short('r').BoolVar(&right)

	// ---------------------------------------------------------------
	// Commands using UID
	filterFolderCmd = app.Command("browse", "Filter the contents of a bookmark folder.").Alias("B")
	openCmd = app.Command("open", "Open bookmark(s) or folder(s).").Alias("o")
	// Common options
	for _, cmd := range []*kingpin.CmdClause{filterFolderCmd, openCmd} {
		cmd.Flag("uid", "Bookmark/folder UID.").Short('u').StringVar(&uid)
	}

	// ---------------------------------------------------------------
	// Commands using query etc.
	filterBookmarksCmd = app.Command("bookmarks", "Filter your bookmarks.").Alias("b")
	filterAllFoldersCmd = app.Command("folders", "Filter your bookmark folders.").Alias("f")
	filterReadingListCmd = app.Command("reading-list", "Filter your Reading List.").Alias("r")
	filterTabsCmd = app.Command("tabs", "Filter your tabs.").Alias("t")
	// Common options
	for _, cmd := range []*kingpin.CmdClause{filterBookmarksCmd, filterFolderCmd, filterAllFoldersCmd, filterReadingListCmd, filterTabsCmd, filterTabActionsCmd, filterURLActionsCmd} {
		cmd.Flag("query", "Search query.").Short('q').StringVar(&query)
		cmd.Flag("max-results", "Maximum number of results to send to Alfred.").
			Short('r').Default(defaultMaxResults).IntVar(&maxResults)
		cmd.Flag("min-score", "Minimum score for search matches.").
			Short('s').Default(defaultMinScore).Float64Var(&minimumScore)
	}
	filterTabsCmd.Flag("max-cache", "Maximum time to cache tab list for.").
		Short('c').Default(defaultMaxCacheAge).DurationVar(&maxCacheAge)

	// ---------------------------------------------------------------
	// Other commands
	activeTabCmd = app.Command("active-tab", "Generate workflow variables for active tab.").Alias("at")
	distnameCmd = app.Command("distname", "Print name for .alfredworkflow file.").Alias("dn")

	app.DefaultEnvars()
}

// --------------------------------------------------------------------
// Actions

// Activate the specified window (and tab).
func doActivate() error {

	log.Printf("Activating %dx%d", window, tab)

	return safari.ActivateTab(window, tab)
}

// Open the bookmark(s)/folder(s) with the specified UIDs.
func doOpen() error {

	if uid == "" {
		log.Println("No UID specified")
		return nil
	}

	invalidateCache()

	log.Printf("Searching for %v ...", uid)

	if bm := safari.BookmarkForUID(uid); bm != nil {
		log.Printf("Opening \"%s\" (%s) ...", bm.Title(), bm.URL)
		return openURL(bm.URL)
	}

	if f := safari.FolderForUID(uid); f != nil {

		errs := []error{}

		for _, bm := range f.Bookmarks {
			log.Printf("Opening \"%s\" (%s) ...", bm.Title(), bm.URL)
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
				SortKey(fmt.Sprintf("%s %s", t.Title, urlKeywords(t.URL)))

			if t.Active {
				it.Icon(IconActive)
			} else {
				it.Icon(IconDefault)
			}

			it.Var("ALSF_WINDOW", fmt.Sprintf("%d", t.WindowIndex)).
				Var("ALSF_TAB", fmt.Sprintf("%d", t.Index)).
				Var("ALSF_URL", t.URL).
				Var("action", "activate")

			it.NewModifier("cmd").
				Subtitle("Other actions…").
				Var("action", "actions")

		}
	}

	if query != "" {
		res := wf.Filter(query)
		log.Printf("%d results for `%s`", len(res), query)
	}
	if wf.IsEmpty() {
		wf.Warn("No tabs found", "Try a different query?")
		return nil
	}
	wf.SendFeedback()

	return nil
}

// Filters all bookmark folders and output Alfred results.
func doFilterAllFolders() error {

	log.Printf("query=%s", query)

	sf := safari.Folders()

	// Send results
	// log.Printf("Sending %d results to Alfred ...", len(ff))
	for _, f := range sf {
		folderItem(f)
	}

	if query != "" {
		res := wf.Filter(query)
		log.Printf("%d folders match `%s`", len(res), query)
	}

	wf.WarnEmpty("No folders found", "Try a different query?")
	wf.SendFeedback()
	return nil
}

// Filters the contents of a specific folder and output Alfred results.
func doFilterFolder() error {

	log.Printf("query=%s, uid=%s", query, uid)

	// ----------------------------------------------------------------
	// Gather results

	f := safari.FolderForUID(uid)
	if f == nil {
		return fmt.Errorf("No folder found with UID: %s", uid)
	}

	log.Printf("%d folders, %d bookmarks in \"%s\"", len(f.Folders), len(f.Bookmarks), f.Title())

	// ----------------------------------------------------------------
	// Show "Back" options if query is empty
	if query == "" {

		if len(f.Ancestors) > 0 {

			p := f.Ancestors[len(f.Ancestors)-1]

			it := wf.NewItem(fmt.Sprintf("Up to \"%s\"", p.Title())).
				Icon(IconUp).
				Valid(true).
				Var("ALSF_UID", p.UID())

			// Alternate action: Go to All Folders
			it.NewModifier("cmd").
				Subtitle("Go back to All Folders").
				Var("action", "top")

			// Default only
			it.Var("action", "browse")
			// it.SetVar("browse_folder", "1")
		} else if uid != "" { // One of the top-level items, e.g. Favorites
			wf.NewItem("Back to All Folders").
				Valid(true).
				Icon(IconHome).
				Var("action", "top")
		}
	}

	// ----------------------------------------------------------------
	// Sort Folders and Bookmarks
	items := []safari.Item{}

	for _, f2 := range f.Folders {
		items = append(items, f2)
	}
	for _, bm := range f.Bookmarks {
		items = append(items, bm)
	}
	log.Printf("%d items in folder `%s`", len(items), f.Title())

	for _, it := range items {
		if bm, ok := it.(*safari.Bookmark); ok {
			bookmarkItem(bm)
		} else if f2, ok := it.(*safari.Folder); ok {
			folderItem(f2)
		} else {
			log.Printf("Could't cast item: %v", it)
		}
	}
	if query != "" {
		res := wf.Filter(query)
		log.Printf("%d results for `%s`", len(res), query)
	} else {
		// TODO: sort items
	}

	wf.WarnEmpty("No bookmarks or folders found", "Try a different query?")
	wf.SendFeedback()

	return nil
}

// Filter bookmarks and outputs Alfred results.
func doFilterBookmarks() error { return filterBookmarks(safari.Bookmarks()) }

// Filter Safari's Reading List and sends results to Alfred.
func doFilterReadingList() error { return filterBookmarks(safari.ReadingList().Bookmarks) }

// filterBookmarks filters bookmarks and outputs Alfred results.
func filterBookmarks(bookmarks []*safari.Bookmark) error {

	log.Printf("query=%s", query)

	log.Printf("Loaded %d bookmarks", len(bookmarks))

	for _, bm := range bookmarks {
		bookmarkItem(bm)
	}

	if query != "" {
		res := wf.Filter(query)
		log.Printf("%d bookmarks for `%s`", len(res), query)
		for i, r := range res {
			log.Printf("#%02d %5.2f `%s`", i+1, r.Score, r.SortKey)
		}
	}

	wf.WarnEmpty("No bookmarks found", "Try a different query?")
	wf.SendFeedback()
	return nil
}

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

func doFilterURLActions() error {
	log.Printf("URL=%s", actionURL)
	LoadScripts(scriptDirs...)
	ua := URLActions()
	acts := make([]Actionable, len(ua))
	for i, a := range ua {
		acts[i] = a
	}
	return listActions(acts)
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

func doFilterTabActions() error {
	LoadScripts(scriptDirs...)
	acts := []Actionable{}
	for _, a := range TabActions() {
		acts = append(acts, a)
	}
	for _, a := range URLActions() {
		acts = append(acts, a)
	}
	return listActions(acts)
}

// doTabAction performs an action on a tab.
func doTabAction() error {
	log.Printf("window=%d, tab=%d, action=%s", window, tab, action)

	LoadScripts(scriptDirs...)
	var ta TabActionable
	var ua URLActionable
	ta = TabAction(action)
	if ta == nil {
		ua = URLAction(action)
		if ua == nil {
			return fmt.Errorf("Unknown action : %s", action)
		}
	}
	// log.Printf("action=%v", a)
	wins, err := loadWindows()
	if err != nil {
		return err
	}
	for _, w := range wins {
		if w.Index == window {
			for _, t := range w.Tabs {
				if t.Index == tab {
					if ta != nil {
						return ta.Run(t)
					}
					u, err := url.Parse(t.URL)
					if err != nil {
						return err
					}
					return ua.Run(u)
				}
			}
		}
	}
	return fmt.Errorf("Tab not found : %02dx%02d", window, tab)
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

// doDistname prints the filename of the .alfredworkflow file to STDOUT.
func doDistname() error {
	fmt.Println(strings.Replace(
		fmt.Sprintf("%s %s.alfredworkflow", wf.Name(), wf.Version()),
		" ", "-", -1))
	return nil
}

// --------------------------------------------------------------------
// Helpers

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
	log.Printf("Saved tab list: %s", aw.ShortenPath(tabCachePath))

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

// listActions sends a list of actions to Alfred.
func listActions(actions []Actionable) error {
	log.Printf("query=%s", query)

	for _, a := range actions {
		it := wf.NewItem(a.Title()).
			Arg(a.Title()).
			Icon(a.Icon()).
			Valid(true).
			Var("ALSF_ACTION", a.Title())

		if _, ok := a.(TabActionable); ok {
			it.Var("ACTION_TYPE", "tab")
		} else if _, ok := a.(URLActionable); ok {
			it.Var("ACTION_TYPE", "url")
		}
	}

	if query != "" {
		res := wf.Filter(query)
		log.Printf("%d actions for `%s`", len(res), query)
	}
	wf.WarnEmpty("No actions found", "Try a different query?")
	wf.SendFeedback()
	return nil
}

// bookmarkItem returns a feedback Item for Safari Bookmark.
func bookmarkItem(bm *safari.Bookmark) *aw.Item {

	it := wf.NewItem(bm.Title()).
		Subtitle(bm.URL).
		Arg(bm.URL).
		UID(bm.UID()).
		Valid(true).
		Copytext(bm.URL).
		Largetype(bm.Preview).
		Var("ALSF_UID", bm.UID()).
		Var("ALSF_URL", bm.URL).
		Var("action", "open")

	it.NewModifier("cmd").
		Subtitle("Other actions…").
		Var("action", "actions")

	if bm.InReadingList() {
		it.Icon(IconReadingList)
	} else {
		it.Icon(IconBookmark)
	}

	return it
}

// folderSubtitle generates a subtitle for a Folder.
func folderSubtitle(f *safari.Folder) string {
	s := []string{}
	for _, f2 := range f.Ancestors {
		s = append(s, f2.Title())
	}
	return strings.Join(s, " / ")
}

// folderTitle generates a title for a Folder.
func folderTitle(f *safari.Folder) string {
	return fmt.Sprintf("%s (%d bookmarks)", f.Title(), len(f.Bookmarks))
}

// folderItem returns a feedback Item for Safari Folder.
func folderItem(f *safari.Folder) *aw.Item {

	it := wf.NewItem(folderTitle(f)).
		Subtitle(folderSubtitle(f)).
		Icon(IconFolder)

	// Make folder actionable if it isn't empty
	if len(f.Bookmarks)+len(f.Folders) > 0 {
		it.Valid(true).
			Var("ALSF_UID", f.UID())

		// Allow opening folder if it contains bookmarks
		m := it.NewModifier("cmd")

		if len(f.Bookmarks) > 0 {

			m.Subtitle(fmt.Sprintf("Open %d bookmark(s)", len(f.Bookmarks)))
			m.Var("action", "open")

		} else {
			m.Valid(false)
		}
		// Default only
		it.Var("action", "browse")
	}

	return it
}

// run is the main script entry point. It's called from main.
func run() {
	var err error

	cmd, err := app.Parse(wf.Args())
	if err != nil {
		wf.FatalError(err)
	}
	wf.MaxResults = maxResults

	switch cmd {

	case activateCmd.FullCommand():
		err = doActivate()

	case filterBookmarksCmd.FullCommand():
		err = doFilterBookmarks()

	case filterFolderCmd.FullCommand():
		err = doFilterFolder()

	case closeCmd.FullCommand():
		err = doClose()

	case filterAllFoldersCmd.FullCommand():
		err = doFilterAllFolders()

	case openCmd.FullCommand():
		err = doOpen()

	case filterReadingListCmd.FullCommand():
		err = doFilterReadingList()

	case filterTabsCmd.FullCommand():
		err = doFilterTabs()

	case distnameCmd.FullCommand():
		err = doDistname()

	case runURLActionCmd.FullCommand():
		err = doURLAction()

	case runTabActionCmd.FullCommand():
		wf.TextErrors = true
		err = doTabAction()

	case filterTabActionsCmd.FullCommand():
		err = doFilterTabActions()

	case filterURLActionsCmd.FullCommand():
		err = doFilterURLActions()

	case activeTabCmd.FullCommand():
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
