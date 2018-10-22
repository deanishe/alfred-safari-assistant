// Copyright (c) 2018 Dean Jackson <deanishe@deanishe.net>
// MIT Licence applies http://opensource.org/licenses/MIT

package main

import (
	"log"

	aw "github.com/deanishe/awgo"
	"github.com/deanishe/go-safari/cloud"
)

func doFilterCloudTabs() error {

	showUpdateStatus()

	if err := LoadScripts(scriptDirs...); err != nil {
		return err
	}

	tabs, err := cloud.Tabs()
	if err != nil {
		return err
	}

	log.Printf("%d cloud tab(s)", len(tabs))

	for _, t := range tabs {
		URLerItem(&cloudTabURLer{t})
	}

	wf.WarnEmpty("No matching tabs found", "Try a different query?")
	wf.SendFeedback()

	return nil
}

type cloudTabURLer struct {
	tab *cloud.Tab
}

func (u *cloudTabURLer) Title() string     { return u.tab.Title }
func (u *cloudTabURLer) URL() string       { return u.tab.URL }
func (u *cloudTabURLer) UID() string       { return u.tab.URL }
func (u *cloudTabURLer) Copytext() string  { return u.tab.URL }
func (u *cloudTabURLer) Largetype() string { return u.tab.URL }
func (u *cloudTabURLer) Icon() *aw.Icon    { return IconCloud }
