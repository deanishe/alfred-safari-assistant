//
// Copyright (c) 2016 Dean Jackson <deanishe@deanishe.net>
//
// MIT Licence. See http://opensource.org/licenses/MIT
//
// Created on 2016-11-04
//

package safari

// Safari automation scripts
const (
	// jsGetCurrentTab -> JSON
	jsGetCurrentTab = `

// ObjC.import('stdlib')

var safari = Application('Safari')
safari.includeStandardAdditions = true


function getCurrentTab() {
  var winIdx = 1,
    tab = safari.windows[0].currentTab,
    tabIdx = tab.index(),
    title = tab.name(),
    url = tab.url()
  console.log('win=1, tab=' + tabIdx + ', title="' + title + '", url=' + url)
  return {title: title, url: url, index: tabIdx, windowIndex: 1}
}

function run(argv) {
  return JSON.stringify(getCurrentTab())
}
`
	// jsGetTabs -> JSON
	jsGetTabs = `

// ObjC.import('stdlib')
// ObjC.import('stdio')

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
`

	// jsActivate <window-number> [<tab-number>] -> nil
	jsActivate = `

ObjC.import('stdlib')

var safari = Application('Safari')
safari.includeStandardAdditions = true

// activateWindow | Activate Safari and bring the specified window to the front.
function activateWindow(winIdx) {
  var win = safari.windows[winIdx-1]()

  if (winIdx != 1) {
    win.visible = false
    win.visible = true
  }

  safari.activate()
}

// activateTab | Activate Safari, bring window to front and make specified tab active.
function activateTab(winIdx, tabIdx) {

  try {
    var win = safari.windows[winIdx-1]()
  }
  catch (e) {
    console.log('Invalid window: ' + winIdx)
    $.exit(1)
  }

  if (tabIdx == 0) { // Activate window
    activateWindow(winIdx)
    return
  }

  // Find tab to activate
  try {
    var tab = win.tabs[tabIdx-1]()
  }
  catch (e) {
    console.log('Invalid tab for window ' + winIdx + ': ' + tabIdx)
    $.exit(1)
  }

  // Activate window and tab if it's not the current tab
  activateWindow(winIdx)
  if (!tab.visible()) {
    win.currentTab = tab
  }

}

// run | CLI entry point
function run(argv) {
  var win = 0,
    tab = 0;

  win = parseInt(argv[0], 10)
  if (argv.length > 1) {
    tab = parseInt(argv[1], 10)
  }

  if (isNaN(win)) {
    console.log('Invalid window: ' + win)
    $.exit(1)
  }

  if (isNaN(tab)) {
    console.log('Invalid tab: ' + tab)
    $.exit(1)
  }

  activateTab(win, tab)
}
`

	jsClose = `
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
`

	// jsRunJavaScript <win> <tab> <js> | Run JavaScript in a tab
	jsRunJavaScript = `

ObjC.import('stdlib')

var safari = Application('Safari')
safari.includeStandardAdditions = true


// runJSInTab <win> <tab> <js> | Run JavaScript in tab
function runJSInTab(winIdx, tabIdx, js) {

  try {
    var win = safari.windows[winIdx-1]()
  }
  catch (e) {
    console.log('Invalid window: ' + winIdx)
    $.exit(1)
  }

  try {
    var tab = win.tabs[tabIdx-1]()
  }
  catch (e) {
    console.log('Invalid tab for window ' + winIdx + ': ' + tabIdx)
    $.exit(1)
  }

  safari.doJavaScript(js, {in: tab})
}

function run(argv) {
  var winIdx = 0,
      tabIdx = 0;

  if (argv.length != 3) {
    console.log('Usage: SafariRunJS.js <win> <tab> <script>')
    $.exit(1)
  }

  winIdx = parseInt(argv[0], 10)
  tabIdx = parseInt(argv[1], 10)
  js = argv[2]

  if (isNaN(winIdx)) {
    console.log('Invalid window: ' + winIdx)
    $.exit(1)
  }
  if (isNaN(tabIdx)) {
    console.log('Invalid tab: ' + tabIdx)
    $.exit(1)
  }

  console.log('Running JS in tab ' + winIdx + 'x' + tabIdx + ' ...')

  runJSInTab(winIdx, tabIdx, js)
}

`
)
