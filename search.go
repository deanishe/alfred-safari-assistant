//
// Copyright (c) 2017 Dean Jackson <deanishe@deanishe.net>
//
// MIT Licence. See http://opensource.org/licenses/MIT
//
// Created on 2017-10-23
//

package main

import (
	"log"
	"time"

	"github.com/deanishe/go-safari"
	"github.com/deanishe/go-safari/history"
)

// doSearch searches bookmarks and recent history.
func doSearch() error {

	showUpdateStatus()

	var (
		bms     []*safari.Bookmark
		entries []*history.Entry
		start   time.Time
	)

	start = time.Now()
	bms = safari.FilterBookmarks(func(bm *safari.Bookmark) bool {
		return !bm.IsBookmarklet()
	})

	log.Printf("loaded %d bookmarks in %v", len(bms), time.Now().Sub(start))

	start = time.Now()

	loadHistory := func() (interface{}, error) {

		var (
			all, entries []*history.Entry
			err          error
			seen         = map[string]bool{}
		)

		all, err = history.Recent(recentHistoryEntries)
		if err != nil {
			return nil, err
		}

		// Filter duplicates
		for _, e := range all {
			if seen[e.URL] {
				continue
			}
			entries = append(entries, e)
			seen[e.URL] = true
		}
		log.Printf("removed %d duplicates from History", len(all)-len(entries))
		return entries, nil
	}

	if err := wf.Session.LoadOrStoreJSON("history", loadHistory, &entries); err != nil {
		return err
	}

	log.Printf("loaded %d history items in %v", len(entries), time.Now().Sub(start))

	for _, bm := range bms {
		URLerItem(&bmURLer{bm})
	}

	for _, e := range entries {
		URLerItem(&hURLer{e})
	}

	if query != "" {
		res := wf.Filter(query)
		log.Printf("%d results for `%s`", len(res), query)
		// for i, r := range res {
		// 	log.Printf("#%02d %5.2f `%s`", i+1, r.Score, r.SortKey)
		// }
	}

	wf.WarnEmpty("No matches found", "Try a different query?")
	wf.SendFeedback()

	return nil
}
