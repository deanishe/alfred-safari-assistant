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

	safari "git.deanishe.net/deanishe/go-safari"
	aw "github.com/deanishe/awgo"
)

// Open the bookmark(s)/folder(s) with the specified UIDs.
func doOpen() error {

	if uid == "" {
		log.Println("No UID specified")
		return nil
	}

	log.Printf("Searching for %v ...", uid)

	if bm := safari.BookmarkForUID(uid); bm != nil {
		if bm.IsBookmarklet() {
			log.Printf("Executing bookmarklet \"%s\" ...", bm.Title())
			return runBookmarklet(bm)
		}
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

// Filter bookmarks and output Alfred results.
func doFilterBookmarks() error {
	return filterBookmarks(safari.FilterBookmarks(func(bm *safari.Bookmark) bool {
		if includeBookmarklets {
			return true
		}
		return !bm.IsBookmarklet()
	}))
}

// Filter bookmarklets and output Alfred results.
func doFilterBookmarklets() error {
	return filterBookmarks(safari.FilterBookmarks(func(bm *safari.Bookmark) bool {
		return bm.IsBookmarklet()
	}))
}

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

// --------------------------------------------------------------------
// Helpers

// bookmarkItem returns a feedback Item for Safari Bookmark.
func bookmarkItem(bm *safari.Bookmark) *aw.Item {

	it := wf.NewItem(bm.Title()).
		Subtitle(bm.URL).
		UID(bm.UID()).
		Valid(true).
		Copytext(bm.URL).
		Var("ALSF_UID", bm.UID()).
		Var("ALSF_URL", bm.URL).
		Var("action", "open")

	if bm.IsBookmarklet() {
		it.Copytext("bkm:" + bm.UID())
	}

	if bm.InReadingList() {
		it.Largetype(bm.Preview)
	}

	// Set actions
	if !bm.IsBookmarklet() {
		it.NewModifier("cmd").
			Subtitle("Other actions…").
			Var("action", "actions")

		// Custom actions
		if bkmActionCtrl != "" {
			it.NewModifier("ctrl").
				Subtitle(bkmActionCtrl).
				Var("action", "url-action").
				Var("ALSF_URL", bm.URL).
				Var("ALSF_ACTION", bkmActionCtrl)
		}
		if bkmActionOpt != "" {
			it.NewModifier("alt").
				Subtitle(bkmActionOpt).
				Var("action", "url-action").
				Var("ALSF_URL", bm.URL).
				Var("ALSF_ACTION", bkmActionOpt)
		}
		if bkmActionFn != "" {
			it.NewModifier("fn").
				Subtitle(bkmActionFn).
				Var("action", "url-action").
				Var("ALSF_URL", bm.URL).
				Var("ALSF_ACTION", bkmActionFn)
		}
		if bkmActionShift != "" {
			it.NewModifier("shift").
				Subtitle(bkmActionShift).
				Var("action", "url-action").
				Var("ALSF_URL", bm.URL).
				Var("ALSF_ACTION", bkmActionShift)
		}
	}

	// Icon
	if bm.IsBookmarklet() {
		it.Icon(IconBookmarklet)
	} else if bm.InReadingList() {
		it.Icon(IconReadingList)
	} else {
		it.Icon(IconBookmark)
	}

	return it
}
