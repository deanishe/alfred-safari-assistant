#!/usr/bin/env osascript -l JavaScript

//
// Copyright (c) 2019 Dean Jackson <deanishe@deanishe.net>
//
// MIT Licence. See http://opensource.org/licenses/MIT
//
// Created on 2019-02-15
//

/***************************************************************
  Open URL in a new Safari window
***************************************************************/

var safari = Application('Safari');
// safari.includeStandardAdditions = true

function run(argv) {
  var url = argv[0],
    doc = safari.Document().make();

  doc.url = url;
}
