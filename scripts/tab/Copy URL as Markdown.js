#!/usr/bin/env osascript -l JavaScript

//
// Copyright (c) 2017 Dean Jackson <deanishe@deanishe.net>
//
// MIT Licence. See http://opensource.org/licenses/MIT
//
// Created on 2017-11-30
//

/***************************************************************
  Copy tab title and URL to pasteboard as a Markdown link
***************************************************************/

ObjC.import('stdlib');
ObjC.import('AppKit');

var safari = Application('Safari');
safari.includeStandardAdditions = true;
var se = Application('System Events');

// getTab <winIdx>, <tabIdx> | Validate window and tab indices and return tab
function getTab(winIdx, tabIdx) {
  // Validate input
  if (isNaN(winIdx)) {
    console.log('invalid window: ' + winIdx);
    $.exit(1);
  }

  if (isNaN(tabIdx)) {
    console.log('invalid tab: ' + tabIdx);
    $.exit(1);
  }

  var tab, win;

  try {
    win = safari.windows[winIdx-1]();
  }
  catch (e) {
    console.log('invalid window: ' + winIdx);
    $.exit(1);
  }

  try {
    tab = win.tabs[tabIdx-1]();
  }
  catch (e) {
    console.log('invalid tab for window ' + winIdx + ': ' + tabIdx);
    $.exit(1);
  }

  return tab;
}

// copyToPasteboard <tab> | Copy tab's title & URL to pasteboard.
function copyToPasteboard(tab) {
  var pboard = $.NSPasteboard.generalPasteboard,
    link = `[${tab.name()}](${tab.url()})`;

  pboard.clearContents;
  pboard.setStringForType($(link), 'public.utf8-plain-text');
}

// run | CLI entry point
function run(argv) {
  var winIdx = parseInt(argv[0], 10),
      tabIdx = parseInt(argv[1], 10),
      tab = getTab(winIdx, tabIdx);

  copyToPasteboard(tab);
}
