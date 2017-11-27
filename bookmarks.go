//
// Copyright (c) 2017 Dean Jackson <deanishe@deanishe.net>
//
// MIT Licence. See http://opensource.org/licenses/MIT
//
// Created on 2017-10-22
//

package main

import (
	"errors"
	"fmt"
	"log"
	"net/url"

	"github.com/deanishe/awgo"
	"github.com/deanishe/go-safari"
)

// Open the bookmark(s)/folder(s) with the specified UIDs.
func doOpen() error {

	if uid == "" {
		return errors.New("No UID specified")
	}

	// If UID is a URL (i.e. History item), open it
	u, err := url.Parse(uid)
	if err == nil && (u.Scheme == "http" || u.Scheme == "https") {
		return openURL(uid)
	}

	// Find item with UID
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

	showUpdateStatus()

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

// bmURLer implements URLer for a Bookmark.
type bmURLer struct {
	bm *safari.Bookmark
}

// Implement URLer
func (b *bmURLer) Title() string { return b.bm.Title() }
func (b *bmURLer) URL() string   { return b.bm.URL }
func (b *bmURLer) UID() string   { return b.bm.UID() }
func (b *bmURLer) Copytext() string {
	if b.bm.IsBookmarklet() {
		return "bkm:" + b.bm.UID()
	}
	return b.bm.URL
}
func (b *bmURLer) Largetype() string {
	if b.bm.InReadingList() {
		return b.bm.Preview
	}
	return b.bm.URL
}
func (b *bmURLer) Icon() *aw.Icon {
	if b.bm.IsBookmarklet() {
		return IconBookmarklet
	}
	if b.bm.InReadingList() {
		return IconReadingList
	}
	return IconBookmark
}

// bookmarkItem returns a feedback Item for Safari Bookmark.
func bookmarkItem(bm *safari.Bookmark) *aw.Item { return URLerItem(&bmURLer{bm}) }
