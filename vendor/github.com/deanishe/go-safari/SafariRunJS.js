#!/usr/bin/env osascript -l JavaScript
//
// Copyright (c) 2017 Dean Jackson <deanishe@deanishe.net>
//
// MIT Licence. See http://opensource.org/licenses/MIT
//
// Created on 2017-10-22
//

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

//  vim: set ft=javascript ts=2 sw=2 tw=80 et :
