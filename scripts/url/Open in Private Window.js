#!/usr/bin/env osascript -l JavaScript

//
// Copyright (c) 2016 Dean Jackson <deanishe@deanishe.net>
//
// MIT Licence. See http://opensource.org/licenses/MIT
//
// Created on 2016-11-05
//

/***************************************************************
  Open URL in private window.
***************************************************************/

ObjC.import('stdlib')

var safari = Application('Safari')
safari.includeStandardAdditions = true
var se = Application('System Events')
// var url = $.getenv('ALSF_URL')

// Ensure Safari is frontmost app
function activateIfNotFrontmost() {
  if (!safari.frontmost()) {
    safari.activate()
    delay(0.2)
  }
}

// Open URL in a Private window
function openPrivate(url) {
  activateIfNotFrontmost()
  se.keystroke('n', {using: ['command down', 'shift down']})
  delay(1.5)
  var doc = safari.windows[0].document
  doc.url = url
}

function run(argv) {
  var url = argv[0]
  console.log('url=' + url)
  openPrivate(url)
}
