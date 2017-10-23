//
// Copyright (c) 2017 Dean Jackson <deanishe@deanishe.net>
//
// MIT Licence. See http://opensource.org/licenses/MIT
//
// Created on 2017-10-22
//

package main

import (
	"log"

	"git.deanishe.net/deanishe/go-safari/history"
	aw "github.com/deanishe/awgo"
)

// doFilterHistory searches Safari history.
func doFilterHistory() error {
	history.MaxSearchResults = maxResults * 10 // allow for lots of duplicates
	wf.MaxResults = maxResults

	entries, err := history.Search(query)
	if err != nil {
		return err
	}

	// Remove duplicates
	var (
		seen   = map[string]bool{}
		unique = []*history.Entry{}
	)
	for _, e := range entries {
		if seen[e.URL] {
			continue
		}
		seen[e.URL] = true
		unique = append(unique, e)
	}
	entries = unique

	log.Printf("%d results for \"%s\"", len(entries), query)

	for _, e := range entries {
		URLerItem(&hURLer{e})
	}

	wf.WarnEmpty("No matching entries found", "Try a different query?")
	wf.SendFeedback()

	return nil
}

type hURLer struct {
	e *history.Entry
}

func (u *hURLer) Title() string     { return u.e.Title }
func (u *hURLer) URL() string       { return u.e.URL }
func (u *hURLer) UID() string       { return u.e.URL }
func (u *hURLer) Copytext() string  { return u.e.URL }
func (u *hURLer) Largetype() string { return u.e.URL }
func (u *hURLer) Icon() *aw.Icon    { return IconHistory }
