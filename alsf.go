//
// Copyright (c) 2016 Dean Jackson <deanishe@deanishe.net>
//
// MIT Licence. See http://opensource.org/licenses/MIT
//
// Created on 2016-05-30
//

// TODO: Other Actions… for URLs (bookmarks)
// TODO: Bookmarklets
// TODO: Script: Open Bookmark/URL in Private Mode
// TODO: Script: Duplicate tab
// TODO: Script: Open URL in Chrome etc.
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

// doTabs filters tabs and outputs Alfred results.
func doTabs() error {

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

// bookmarkItem returns a feedback Item for Safari Bookmark.
func bookmarkItem(bm *safari.Bookmark) *aw.Item {

	it := wf.NewItem(bm.Title()).
		Subtitle(bm.URL).
		Arg(bm.URL).
		UID(bm.UID()).
		Valid(true).
		Copytext(bm.URL).
		Largetype(bm.Preview)

	if bm.InReadingList() {
		it.Icon(IconReadingList)
	} else {
		it.Icon(IconBookmark)
	}

	it.Var("ALSF_UID", bm.UID()).
		Var("action", "open")

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
			// m.SetVar("open_bookmark", "1")
			m.Var("action", "open")

		} else {
			m.Valid(false)
		}
		// Default only
		// it.SetVar("browse_folder", "1")
		it.Var("action", "browse")
	}

	return it
}

// doFolders filters bookmark folders and outputs Alfred results.
func doFolders() error {

	log.Printf("query=%s", query)

	sf := safari.Folders()
	// ff := make(fuzzyFolders, len(sf))

	// for i, f := range sf {
	// 	ff[i] = f
	// }

	// if query != "" {
	// 	scores := fuzzy.Sort(ff, query)
	// 	for i, s := range scores {
	// 		if s < minimumScore {
	// 			ff = ff[:i]
	// 			break
	// 		}
	// 		if i == maxResults {
	// 			log.Printf("Reached max. results (%d)", maxResults)
	// 			ff = ff[:i]
	// 			break
	// 		}
	// 	}
	// 	log.Printf("%d folders match \"%s\"", len(ff), query)
	// }

	// if len(ff) == 0 {
	// 	wf.Warn("No folders found", "Try a different query?")
	// 	return nil
	// }

	// Send results
	// log.Printf("Sending %d results to Alfred ...", len(ff))
	for _, f := range sf {
		folderItem(f)
	}

	if query != "" {
		res := wf.Filter(query)
		log.Printf("%d folders match `%s`", len(res), query)
	}
	if wf.IsEmpty() {
		wf.Warn("No folders found", "Try a different query?")
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
	tmap := map[string]string{}

	for _, f2 := range f.Folders {
		// fMap[f2.UID] = f2
		tmap[f2.UID()] = "f"
		// proxies = append(proxies, &fuzzyProxy{f2.UID, f2.Title})
		items = append(items, f2)
	}
	for _, bm := range f.Bookmarks {
		// bMap[bm.UID] = bm
		tmap[bm.UID()] = "b"
		// proxies = append(proxies, &fuzzyProxy{bm.UID, bm.Title})
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

func doListURLActions() error {
	LoadScripts(scriptDirs...)
	ua := URLActions()
	acts := make([]Actionable, len(ua))
	for i, a := range ua {
		acts[i] = a
	}
	return listActions(acts)
}

func doListTabActions() error {
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

// run is the main script entry point. It's called from main.
func run() {
	var err error

	cmd, err := app.Parse(os.Args[1:])
	if err != nil {
		wf.FatalError(err)
	}
	wf.MaxResults = maxResults

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
		wf.TextErrors = true
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
