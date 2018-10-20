#!/usr/bin/env osascript -l JavaScript

//
// Copyright (c) 2016 Dean Jackson <deanishe@deanishe.net>
//
// MIT Licence. See http://opensource.org/licenses/MIT
//
// Created on 2016-11-04
//

/***************************************************************
  Simple command-line program to list open tabs in Safari.
***************************************************************/

ObjC.import('stdlib')

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
