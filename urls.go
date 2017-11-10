//
// Copyright (c) 2017 Dean Jackson <deanishe@deanishe.net>
//
// MIT Licence. See http://opensource.org/licenses/MIT
//
// Created on 2017-10-23
//

package main

import (
	"net/url"

	aw "github.com/deanishe/awgo"
)

// URLer is an item with a URL.
type URLer interface {
	Title() string
	URL() string
	UID() string
	Copytext() string
	Largetype() string
	Icon() *aw.Icon
}

// URLerItem returns a feedback Item for a URLer.
func URLerItem(u URLer) *aw.Item {

	it := wf.NewItem(u.Title()).
		Subtitle(u.URL()).
		UID(u.UID()).
		Valid(true).
		Copytext(u.Copytext()).
		Largetype(u.Largetype()).
		Icon(u.Icon()).
		Var("ALSF_UID", u.UID()).
		Var("ALSF_URL", u.URL()).
		Var("action", "open")

	URL, err := url.Parse(u.URL())
	if err == nil {

		if searchHostnames {
			// Add hostname to search keys
			it.Match(u.Title() + " " + URL.Hostname())
		}

		if URL.Scheme == "http" || URL.Scheme == "https" {

			it.NewModifier("cmd").
				Subtitle("Other actions…").
				Var("action", "actions")

			// Custom actions
			actions := []struct{ action, key string }{
				{urlActionFn, "fn"},
				{urlActionCtrl, "ctrl"},
				{urlActionOpt, "alt"},
				{urlActionShift, "shift"},
			}

			for _, a := range actions {
				if a.action == "" { // unset
					continue
				}
				it.NewModifier(a.key).
					Subtitle(a.action).
					Var("action", "url-action").
					Var("ALSF_ACTION", a.action)
			}
		}
	}
	return it
}
