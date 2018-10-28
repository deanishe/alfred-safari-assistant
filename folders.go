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
	"strings"

	"github.com/deanishe/awgo"
	"github.com/deanishe/go-safari"
)

// Filters all bookmark folders and output Alfred results.
func doFilterAllFolders() error {

	showUpdateStatus()

	log.Printf("query=%s", query)

	sf := safari.Folders()

	// Send results
	// log.Printf("Sending %d results to Alfred ...", len(ff))
	for _, f := range sf {
		folderItem(f)
	}

	if query != "" {
		res := wf.Filter(query)
		log.Printf("%d folder(s) match %q", len(res), query)
		for i, r := range res {
			log.Printf("#%02d %5.2f %q", i+1, r.Score, r.SortKey)
		}
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

		wf.Configure(aw.SuppressUIDs(true))

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
	log.Printf("%d item(s) in folder %q", len(items), f.Title())

	for _, it := range items {
		if bm, ok := it.(*safari.Bookmark); ok {
			URLerItem(&bmURLer{bm})
		} else if f2, ok := it.(*safari.Folder); ok {
			folderItem(f2)
		} else {
			log.Printf("Could't cast item: %v", it)
		}
	}
	if query != "" {
		res := wf.Filter(query)
		log.Printf("%d result(s) for %q", len(res), query)
		for i, r := range res {
			log.Printf("#%02d %5.2f %q", i+1, r.Score, r.SortKey)
		}
	}

	wf.WarnEmpty("No bookmarks or folders found", "Try a different query?")
	wf.SendFeedback()

	return nil
}

// --------------------------------------------------------------------
// Helpers

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
		Match(f.Title()).
		UID(f.UID()).
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
