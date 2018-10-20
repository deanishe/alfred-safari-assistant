#!/usr/bin/env osascript -l JavaScript

//
// Copyright (c) 2016 Dean Jackson <deanishe@deanishe.net>
//
// MIT Licence. See http://opensource.org/licenses/MIT
//
// Created on 2016-05-29
//

/***************************************************************
  Simple command-line program to list open tabs in Safari.
***************************************************************/

ObjC.import('stdlib')
ObjC.import('stdio')

function getWindows() {

  var safari = Application('Safari')
  safari.includeStandardAdditions = true

  var results = []
  var wins = safari.windows

  for (i=0; i<wins.length; i++) {
    var data = {'index': i+1, 'tabs': []},
      w = wins[i],
      tabs = w.tabs

    // Ignore non-browser windows
    try {
      data['activeTab'] = w.currentTab().index()
    }
    catch (e) {
      console.log('Ignoring window ' + (i+1))
      continue
    }

    // Tabs
    for (j=0; j<tabs.length; j++) {
      var t = tabs[j]
      data.tabs.push({
        'title': t.name(),
        'url': t.url(),
        'index': j+1,
        'windowIndex': i+1,
	'active': j+1 === data['activeTab']
      })
    }

    results.push(data)
  }
  return results
}

function run(argv) {
  return JSON.stringify(getWindows())
}