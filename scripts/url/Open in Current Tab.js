#!/usr/bin/env osascript -l JavaScript

//
// Copyright (c) 2017 Dean Jackson <deanishe@deanishe.net>
//
// MIT Licence. See http://opensource.org/licenses/MIT
//
// Created on 2017-12-27
//

/***************************************************************
  Open URL in the active tab
***************************************************************/

var safari = Application('Safari')
// safari.includeStandardAdditions = true

function run(argv) {
  var url = argv[0],
    tab = safari.windows[0].currentTab
  tab.url = url
}