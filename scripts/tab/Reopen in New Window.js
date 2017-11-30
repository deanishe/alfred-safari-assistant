#!/usr/bin/env osascript -l JavaScript

//
// Copyright (c) 2017 Dean Jackson <deanishe@deanishe.net>
//
// MIT Licence. See http://opensource.org/licenses/MIT
//
// Created on 2017-11-30
//

/***************************************************************
  Reopen URL in a new window and close active tab
***************************************************************/

ObjC.import('stdlib')

var safari = Application('Safari')
safari.includeStandardAdditions = true
var se = Application('System Events')

// getTab <winIdx>, <tabIdx> | Validate window and tab indices and return tab
function getTab(winIdx, tabIdx) {
  // Validate input
  if (isNaN(winIdx)) {
    console.log('invalid window: ' + winIdx)
    $.exit(1)
  }

  if (isNaN(tabIdx)) {
    console.log('invalid tab: ' + tabIdx)
    $.exit(1)
  }

	try {
    var win = safari.windows[winIdx-1]()
  }
  catch (e) {
    console.log('invalid window: ' + winIdx)
    $.exit(1)
  }

  try {
    var tab = win.tabs[tabIdx-1]()
  }
  catch (e) {
    console.log('invalid tab for window ' + winIdx + ': ' + tabIdx)
    $.exit(1)
  }

  return tab
}

// reopenTab <tab> | Close tab and open its URL in a new window.
function reopenTab(tab) {
  var url = tab.url(),
      doc = safari.Document().make()

  tab.close()
  doc.url = url
}

// run | CLI entry point
function run(argv) {
  var winIdx = parseInt(argv[0], 10),
      tabIdx = parseInt(argv[1], 10),
      tab = getTab(winIdx, tabIdx)

  reopenTab(tab)
}
