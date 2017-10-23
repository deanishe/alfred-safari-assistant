//
// Copyright (c) 2016 Dean Jackson <deanishe@deanishe.net>
//
// MIT Licence. See http://opensource.org/licenses/MIT
//
// Created on 2016-05-30
//

// TODO: Allow user to configure URL/tab actions for other modifiers
// TODO: URL actions for history items
// TODO: Script: Open Bookmark/URL in Private Mode
// TODO: iCloud tabs (~/Library/SyncedPreferences/com.apple.Safari.plist)

package main // import "git.deanishe.net/deanishe/alfred-safari-assistant"

import (
	"fmt"
	"log"
	"net/url"
	"os/exec"
	"path/filepath"
	"strings"

	"git.deanishe.net/deanishe/go-safari"
	"github.com/deanishe/awgo"
	"github.com/deanishe/awgo/util"
	"github.com/juju/deputy"
	"gopkg.in/alecthomas/kingpin.v2"
)

// Defaults for Kingpin flags
const (
	defaultMaxResults = "100"
)

// Icons
var (
	IconDefault     = &aw.Icon{Value: "icon.png"}
	IconTab         = &aw.Icon{Value: "icons/tab.png"}
	IconActive      = &aw.Icon{Value: "icons/tab-active.png"}
	IconReadingList = &aw.Icon{Value: "icons/reading-list.png"}
	IconBookmark    = &aw.Icon{Value: "icons/bookmark.png"}
	IconBookmarklet = &aw.Icon{Value: "icons/bookmarklet.png"}
	IconHistory     = &aw.Icon{Value: "icons/history.png"}
	IconURL         = &aw.Icon{Value: "icons/url.png"}
	IconFolder      = &aw.Icon{Value: "icons/folder.png"}
	IconUp          = &aw.Icon{Value: "icons/up.png"}
	IconHome        = &aw.Icon{Value: "icons/home.png"}
	IconWarning     = &aw.Icon{Value: "icons/warning.png"}
	// IconError       = &aw.Icon{Value: "icons/error.png"}
)

var (
	// Kingpin and script options
	app *kingpin.Application

	// Application commands
	activateCmd, filterBookmarksCmd           *kingpin.CmdClause
	filterBookmarkletsCmd, filterFolderCmd    *kingpin.CmdClause
	filterAllFoldersCmd, filterReadingListCmd *kingpin.CmdClause
	openCmd, closeCmd, filterTabsCmd          *kingpin.CmdClause
	distnameCmd, runActionCmd, searchCmd      *kingpin.CmdClause
	runTabActionCmd, runURLActionCmd          *kingpin.CmdClause
	filterActionsCmd, filterTabActionsCmd     *kingpin.CmdClause
	filterURLActionsCmd, activeTabCmd         *kingpin.CmdClause
	filterHistoryCmd                          *kingpin.CmdClause

	// Script options (populated by Kingpin application)
	query                       string
	left, right                 bool
	winIdx, tabIdx              int
	action, actionType, uid     string
	includeBookmarklets         bool
	actionURL                   *url.URL
	maxResults                  int
	recentHistoryEntries        int
	tabActionOpt, tabActionCtrl string
	tabActionFn, tabActionShift string
	urlActionOpt, urlActionCtrl string
	urlActionFn, urlActionShift string
	// historyLimit                int
	// historyFuzzy                bool

	// Workflow stuff
	wf         *aw.Workflow
	scriptDirs []string

	urlKillWords = []string{"www.", ".com", ".net", ".org", ".co.uk"}
)

// Mostly sets up kingpin commands
func init() {

	// safari.Configure(safari.IgnoreBookmarklets(true))
	// Override default icons
	// aw.IconError = IconError
	aw.IconWarning = IconWarning

	wf = aw.New()
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
	for _, cmd := range []*kingpin.CmdClause{runURLActionCmd, filterURLActionsCmd, filterTabActionsCmd} {
		cmd.Flag("url", "URL to action.").Short('u').Required().URLVar(&actionURL)
	}
	// Common action options
	for _, cmd := range []*kingpin.CmdClause{runTabActionCmd, runURLActionCmd} {
		cmd.Flag("action", "Action name.").Short('a').PlaceHolder("NAME").Required().StringVar(&action)
	}

	runTabActionCmd.Flag("action-type", "Action type.").PlaceHolder("TYPE").Required().StringVar(&actionType)

	// ---------------------------------------------------------------
	// Commands using window and tab
	activateCmd = app.Command("activate", "Active a specific window or tab.").Alias("a")
	closeCmd = app.Command("close", "Close tab(s).").Alias("c")

	// Common options
	for _, cmd := range []*kingpin.CmdClause{activateCmd, closeCmd, runTabActionCmd, filterTabActionsCmd} {
		cmd.Flag("window", "Window number.").
			Short('w').Default("1").IntVar(&winIdx)
		cmd.Flag("tab", "Tab number.").
			Short('t').Required().IntVar(&tabIdx)
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
	searchCmd = app.Command("search", "Filter your bookmarks and recent history.").Alias("s")
	filterBookmarksCmd = app.Command("bookmarks", "Filter your bookmarks.").Alias("b")
	filterBookmarkletsCmd = app.Command("bookmarklets", "Filter your bookmarklets.").Alias("B")
	filterAllFoldersCmd = app.Command("folders", "Filter your bookmark folders.").Alias("f")
	filterReadingListCmd = app.Command("reading-list", "Filter your Reading List.").Alias("r")
	filterTabsCmd = app.Command("tabs", "Filter your tabs.").Alias("t")
	filterHistoryCmd = app.Command("history", "Filter your history.").Alias("h")

	// Common options
	for _, cmd := range []*kingpin.CmdClause{
		filterBookmarksCmd, filterBookmarkletsCmd, filterFolderCmd,
		filterAllFoldersCmd, filterReadingListCmd, filterTabsCmd,
		filterTabActionsCmd, filterURLActionsCmd, filterHistoryCmd,
		searchCmd,
	} {
		cmd.Flag("query", "Search query.").Short('q').StringVar(&query)
		cmd.Flag("max-results", "Maximum number of results to send to Alfred.").
			Short('r').Default(defaultMaxResults).IntVar(&maxResults)
	}

	// ---------------------------------------------------------------
	// Options set via workflow configuration sheet
	filterBookmarksCmd.Flag("include-bookmarklets", "Include bookmarklets with bookmarks.").
		BoolVar(&includeBookmarklets)

	// filterHistoryCmd.Flag("fuzzy-history-limit", "The maximum number of history items to load.").
	// 	PlaceHolder("NUMBER").
	// 	IntVar(&historyLimit)
	// filterHistoryCmd.Flag("fuzzy-history-search", "Use fuzzy search for history.").
	// 	BoolVar(&historyFuzzy)
	searchCmd.Flag("history-entries", "Number of recent history entries to load.").
		IntVar(&recentHistoryEntries)

	for _, cmd := range []*kingpin.CmdClause{
		filterBookmarksCmd, filterReadingListCmd,
		filterHistoryCmd, searchCmd,
	} {

		// Alternate URL actions
		cmd.Flag("url-ctrl", "Action to run for CTRL key.").
			PlaceHolder("SCRIPT_NAME").
			StringVar(&urlActionCtrl)
		cmd.Flag("url-opt", "Action to run for OPT (ALT) key.").
			PlaceHolder("SCRIPT_NAME").
			StringVar(&urlActionOpt)
		cmd.Flag("url-fn", "Action to run for FN key.").
			PlaceHolder("SCRIPT_NAME").
			StringVar(&urlActionFn)
		cmd.Flag("url-shift", "Action to run for SHIFT key.").
			PlaceHolder("SCRIPT_NAME").
			StringVar(&urlActionShift)
	}
	// Alternate tab actions
	filterTabsCmd.Flag("tab-ctrl", "Action/bookmarklet to run for CTRL key.").
		PlaceHolder("SCRIPT_NAME").
		StringVar(&tabActionCtrl)
	filterTabsCmd.Flag("tab-opt", "Action/bookmarklet to run for OPT (ALT) key.").
		PlaceHolder("SCRIPT_NAME").
		StringVar(&tabActionOpt)
	filterTabsCmd.Flag("tab-fn", "Action/bookmarklet to run for FN key").
		PlaceHolder("SCRIPT_NAME").
		StringVar(&tabActionFn)
	filterTabsCmd.Flag("tab-shift", "Action/bookmarklet to run for SHIFT key.").
		PlaceHolder("SCRIPT_NAME").
		StringVar(&tabActionShift)

	// ---------------------------------------------------------------
	// Other commands
	activeTabCmd = app.Command("active-tab", "Generate workflow variables for active tab.").Alias("at")
	distnameCmd = app.Command("distname", "Print name for .alfredworkflow file.").Alias("dn")

	app.DefaultEnvars()
}

// --------------------------------------------------------------------
// Actions

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
	wf.TextErrors = true

	log.Printf("URL=%s, action=%s", actionURL, action)

	LoadScripts(scriptDirs...)

	a := URLAction(action)
	if a == nil {
		return fmt.Errorf("Unknown action : %s", action)
	}
	return a.Run(actionURL)
}

// doDistname prints the filename of the .alfredworkflow file to STDOUT.
func doDistname() error {
	fmt.Print(strings.Replace(
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

// loadWindows returns a list of Safari windows and caches them for the duration of the session.
func loadWindows() ([]*safari.Window, error) {

	var wins []*safari.Window

	getWins := func() (interface{}, error) {
		return safari.Windows()
	}

	if err := wf.Session.LoadOrStoreJSON("windows", getWins, &wins); err != nil {
		return nil, err
	}
	return wins, nil
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
			Copytext(a.Title()).
			Valid(true).
			Var("action", "tab-action").
			Var("ALSF_ACTION", a.Title())

		if _, ok := a.(TabActionable); ok {
			it.Var("ALSF_ACTION_TYPE", "tab")
		} else if _, ok := a.(URLActionable); ok {
			it.Var("ALSF_ACTION_TYPE", "url")
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

// run is the main script entry point. It's called from main.
func run() {
	var err error

	cmd, err := app.Parse(wf.Args())
	if err != nil {
		wf.FatalError(err)
	}
	wf.MaxResults = maxResults

	// Create user script directories
	util.MustExist(filepath.Join(wf.DataDir(), "scripts", "tab"))
	util.MustExist(filepath.Join(wf.DataDir(), "scripts", "url"))

	switch cmd {

	case activateCmd.FullCommand():
		err = doActivate()

	case filterBookmarksCmd.FullCommand():
		err = doFilterBookmarks()

	case filterBookmarkletsCmd.FullCommand():
		err = doFilterBookmarklets()

	case filterFolderCmd.FullCommand():
		err = doFilterFolder()

	case filterAllFoldersCmd.FullCommand():
		err = doFilterAllFolders()

	case filterHistoryCmd.FullCommand():
		err = doFilterHistory()

	case filterReadingListCmd.FullCommand():
		err = doFilterReadingList()

	case filterTabsCmd.FullCommand():
		err = doFilterTabs()

	case filterTabActionsCmd.FullCommand():
		err = doFilterTabActions()

	case filterURLActionsCmd.FullCommand():
		err = doFilterURLActions()

	case searchCmd.FullCommand():
		err = doSearch()

	case closeCmd.FullCommand():
		err = doClose()

	case openCmd.FullCommand():
		err = doOpen()

	case distnameCmd.FullCommand():
		err = doDistname()

	case runURLActionCmd.FullCommand():
		err = doURLAction()

	case runTabActionCmd.FullCommand():
		wf.TextErrors = true
		err = doTabAction()

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
