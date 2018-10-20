#!/usr/bin/env osascript -l JavaScript
//
// Copyright (c) 2016 Dean Jackson <deanishe@deanishe.net>
//
// MIT Licence. See http://opensource.org/licenses/MIT
//
// Created on 2016-05-30
//

Array.prototype.contains = function(o) {
  return this.indexOf(o) > -1
}

ObjC.import('stdlib');

// Permissible targets
var whats = ['win', 'tab', 'tabs-other', 'tabs-left', 'tabs-right'],
  app = Application('Safari');
  app.includeStandardAdditions = true;

// usage | Print help to STDOUT
function usage() {

  console.log('SafariClose.js (win|tab|tabs-other|tabs-left|tabs-right) [<win>] [<tab>]');
  console.log('');
  console.log('Close specified window and/or tab(s). If not specified, <win> and <tab>');
  console.log('default to the frontmost window and current tab respectively.');
  console.log('');
  console.log('Usage:');
  console.log('    SafariClose.js win [<win>]');
  console.log('    SafariClose.js (tab|tabs-other|tabs-left|tabs-right) [<win>] [<tab>]');
  console.log('    SafariClose.js -h');
  console.log('');
  console.log('Options:');
  console.log('    -h    Show this help message and exit.');

}

// closeWindow | Close the specified Safari window
function closeWindow(winIdx) {
  var win = app.windows[winIdx-1];
  win.close();
}

// getCurrentTab | Return the index of the current tab of frontmost window
function getCurrentTab() {
  return app.windows[0].currentTab.index()
}

// closeTabs | tabFunc(idx, tab) is called for each tab in the window.
// Tab is closed if it returns true.
function closeTabs(winIdx, tabFunc) {

  var win = app.windows[winIdx-1],
      tabs = win.tabs,
      current = win.currentTab,
      toClose = [];

  // Loop backwards, so tab indices don't change as we close them
  for (i=tabs.length-1; i>-1; i--) {
    var tab = tabs[i];
    if (tabFunc(i+1, tab)) {
      console.log('Closing tab ' + (i+1) + ' ...');
      tab.close();
    }
  }

}

function run(argv) {
  var what = argv[0],
      winIdx = 1,  // default to frontmost window
      tabIdx = 0;

  if (argv.contains('-h') || argv.contains('--help')) {
    usage();
    $.exit(0);
  }

  // Validate arguments
  if (!whats.contains(what)) {
    console.log('Invalid target: ' + what);
    console.log('');
    usage();
    $.exit(1);
  }

  if (typeof(argv[1]) != 'undefined') {
    winIdx = parseInt(argv[1], 10);
    if (isNaN(winIdx)) {
      console.log('Invalid window number: ' + argv[1]);
      $.exit(1);
    }
  }

  if (what != 'win') {
    if (typeof(argv[2]) != 'undefined') {
      var tabIdx = parseInt(argv[2], 10);
      if (isNaN(tabIdx)) {
        console.log('Invalid tab number for window ' + winIdx + ': ' + argv[2]);
        $.exit(1);
      }
    } else {
      tabIdx = getCurrentTab();
    }
  }

  console.log('winIdx=' + winIdx + ', tabIdx=' + tabIdx);

  // Let's close some shit
  if (what == 'win') {

    return closeWindow(winIdx)

  } else if (what == 'tab') {

    //return closeTab(winIdx, tabIdx)
    return closeTabs(winIdx, function(i, t) {
      return i === tabIdx
    })

  } else if (what == 'tabs-other') {

    return closeTabs(winIdx, function(i, t) {
      return i != tabIdx
    })

  } else if (what == 'tabs-left') {

    return closeTabs(winIdx, function(i, t) {
      return i < tabIdx
    })


  } else if (what == 'tabs-right') {

    return closeTabs(winIdx, function(i, t) {
      return i > tabIdx
    })

  }
}