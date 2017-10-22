
Alfred Safari
=============

Search and open/activate your Safari bookmark(let)s and tabs from Alfred 3.

Includes several actions for tabs/bookmarks and allows you to add your own via scripts.

Usage
=====

- `bm [<query>]` — Search and open/action bookmarks.
    - `↩` — Open bookmark.
    - `⌘↩` — Show URL actions.
- `bml [<query>]` — Search and run bookmarklets.
    - `↩` — Run bookmarklet in active tab.
- `bmf [<query>]` — Search bookmark folders.
    - `↩` — Enter folder/open bookmark.
    - `⌘↩` — Open all bookmarks in folder/show URL actions for bookmark.
- `rl [<query>]` — Search and open/action Reading List entries.
- `tab [<query>]` — Search and activate/action Safari tabs.
    - `↩` — Activate the selected tab.
    - `⌘↩` — Show tab actions for selected tab.



Configuration
=============

There is a single setting in the workflow's configuration sheet: `ALSF_INCLUDE_BOOKMARKLETS`. Set this to `1` to include bookmarklets in the normal bookmark search (`bm`).


Action scripts
==============

Much of the workflow's functionality is implemented via built-in scripts. You can also add your own scripts to provide additional tab and/or URL actions.

Add your own to the workflow's data directory. Use the magic command `workflow:data` to open the data directory, e.g. `bm workflow:data`.

Scripts go in a subdirectory of the `scripts` directory depending on the type. Tab scripts go in `scripts/tab`, URL scripts in `scripts/url`.

When you view actions for a Safari tab, both tab and URL actions are listed. When you action a bookmark, only URL actions are listed.

Tab scripts are called with the indices of the selected window and tab as `$1` and `$2`. So if the third tab of the second Safari window is active, your script is called as `/path/to/script 2 3`.

URL scripts are called with the URL of the selected bookmark or tab as `$1`, e.g. `/path/to/script http://www.example.com`.

See the built-in scripts (in the `scripts` subdirectory of the workflow) for examples of how to implement them.


### Supported languages

The workflow knows to run `.scpt`, `.js`, `.applescript` and `.scptd` scripts via `/usr/bin/osascript`. It can also run any script/program with its executable bit set (it will call these directly).


### Script icons

By default, tab scripts get a tab icon and URL scripts a URL one. You can supply a custom icon for any script by saving the icon alongside the script with the same basename (i.e. the same name as the script, only with a different file extension). Supported icon extensions are `.png`, `.icns`, `.jpg`, `.jpeg` and `.gif`.


### Built-in actions

The following actions are built into the workflow, either hard-coded or as bundled scripts.


#### Tab actions

These actions are available for tabs only.

- Close Tab
- Close Window
- Close Tabs to Left
- Close Tabs to Right


#### URL actions

These actions are available for bookmarks and tabs (that have URLs).

- Open URL in Default Browser
- Open in Chrome
- Open in Firefox
- Open in Private Window

