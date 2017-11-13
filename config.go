//
// Copyright (c) 2017 Dean Jackson <deanishe@deanishe.net>
//
// MIT Licence. See http://opensource.org/licenses/MIT
//
// Created on 2017-11-02
//

package main

import "path/filepath"

var (
	issuesURL = "https://github.com/deanishe/alfred-safari-assistant/issues"
	forumURL  = "https://www.alfredforum.com/topic/10921-safari-assistant/"
)

// Show configuration options in Alfred.
func doConfig() error {

	blPath, err := initBlacklist()
	if err != nil {
		return err
	}

	wf.NewItem("View Help File").
		Subtitle("Open the help file in your browser").
		Arg("./README.html").
		Valid(true).
		Icon(IconHelp).
		Var("action", "open")

	wf.NewItem("Edit Action Blacklist").
		Subtitle("Open action blacklist in your editor").
		Arg(blPath).
		Valid(true).
		Icon(IconBlacklistEdit).
		Var("action", "open")

	wf.NewItem("User Scripts").
		Subtitle("Open user scripts directory in Finder").
		Arg(filepath.Join(wf.DataDir(), "scripts")).
		Valid(true).
		Icon(IconFolder).
		Var("action", "open")

	wf.NewItem("Check for Update").
		Subtitle("Check to see if a new version is available").
		Valid(false).
		Icon(IconUpdateCheck).
		Autocomplete("workflow:update")

	wf.NewItem("Report Problem on GitHub").
		Subtitle("Open the workflow's issue tracker in your browser").
		Arg(issuesURL).
		Valid(true).
		Icon(IconGitHub).
		Var("action", "open")

	wf.NewItem("Visit Forum Thread").
		Subtitle("Open workflow thread on alfredforum.com").
		Arg(forumURL).
		Valid(true).
		Icon(IconURL).
		Var("action", "open")

	if query != "" {
		wf.Filter(query)
	}

	wf.WarnEmpty("No matching items", "Try a different query?")

	wf.SendFeedback()

	return nil
}
