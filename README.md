
<img src="./icons/icon.png" width="128" height="128">

Alfred Safari
=============

Search and open/activate your Safari bookmark(let)s and tabs from Alfred 3.

Includes several actions for tabs/bookmarks and allows you to add your own via scripts. Assign your favourite actions (and bookmarklets) to hotkeys.

<!-- MarkdownTOC autolink="true" bracket="round" depth="2" autoanchor="true" -->

- [Usage](#usage)
- [Configuration](#configuration)
- [Action scripts](#action-scripts)
    - [Supported languages](#supported-languages)
    - [Script icons](#script-icons)
    - [Built-in actions](#built-in-actions)
- [History](#history)
- [Licensing & thanks](#licensing--thanks)

<!-- /MarkdownTOC -->


<a name="usage"></a>
Usage
-----

- `bh [<query>]` — Search and open/action bookmarks and recent history. (See [History](#history) section below.)
    - `↩` — Open item in browser.
    - `⌘↩` — Show URL actions for selected item.
    - `⌥↩` — Run custom action on selected item.
    - `^↩` — Run custom action on selected item.
    - `fn↩` — Run custom action on selected item.
    - `⇧↩` — Run custom action on selected item.
- `bm [<query>]` — Search and open/action bookmarks.
    - `↩`, `⌘↩`, `⌥↩`, `^↩`, `fn↩`, `⇧↩` — As above.
- `bml [<query>]` — Search and run bookmarklets.
    - `↩` — Run bookmarklet in active tab.
- `bmf [<query>]` — Search bookmark folders.
    - `↩` — Enter folder/open bookmark.
    - `⌘↩` — Open all bookmarks in folder/show URL actions for bookmark.
- `hi [<query>]` — Search and open/action history entries. (See [History](#history) section below.)
    - `↩`, `⌘↩`, `⌥↩`, `^↩`, `fn↩`, `⇧↩` — As above.
- `rl [<query>]` — Search and open/action Reading List entries.
    - `↩`, `⌘↩`, `⌥↩`, `^↩`, `fn↩`, `⇧↩` — As above.
- `tab [<query>]` — Search and activate/action Safari tabs.
    - `↩` — Activate the selected tab.
    - `⌘↩`, `⌥↩`, `^↩`, `fn↩`, `⇧↩` — As above.



<a name="configuration"></a>
Configuration
-------------

There are several settings in the workflow's configuration sheet:

- `ALSF_HISTORY_ENTRIES`. Number of recent history entries to load for `bh` action (search bookmarks and recent history).
- `ALSF_INCLUDE_BOOKMARKLETS`. Set this to `1` to include bookmarklets in the normal bookmark search (`bm`).

The following settings assign actions or bookmarklets to modifier keys:

| Key                | Action                                         |
| ------------------ | ---------------------------------------------- |
| `ALSF_TAB_CTRL`    | `^↩` custom action/bookmarklet for tab         |
| `ALSF_TAB_OPT`     | `⌥↩` custom action/bookmarklet for tab         |
| `ALSF_TAB_FN`      | `fn↩` custom action/bookmarklet for tab        |
| `ALSF_TAB_SHIFT`   | `⇧↩` custom action/bookmarklet for tab         |
| `ALSF_URL_CTRL`    | `^↩` custom action for bookmark/history entry  |
| `ALSF_URL_OPT`     | `⌥↩` custom action for bookmark/history entry  |
| `ALSF_URL_FN`      | `fn↩` custom action for bookmark/history entry |
| `ALSF_URL_SHIFT`   | `⇧↩` custom action for bookmark/history entry  |

The `ALSF_TAB_*` variables assign custom actions or bookmarklets available when browsing Safari tabs. The `ALSF_URL_*` variables assign custom actions (*not* bookmarklets) to bookmarks and history entries.

To assign a script action, enter the corresponding script's name (without extension) as the value for the variable. To assign a bookmarklet, use `bkm:<UID>` where `<UID>` is the bookmarklet's UID.

In either case, press `⌘C` on an action or bookmarklet in Alfred's UI to copy the corresponding value, then paste it into the configuration sheet as the value for the appropriate variable.


<a name="action-scripts"></a>
Action scripts
--------------

Much of the workflow's functionality is implemented via built-in scripts. You can also add your own scripts to provide additional tab and/or URL actions.

Add your own to the workflow's data directory. Use the magic command `workflow:data` to open the data directory, e.g. `bm workflow:data`.

Scripts go in a subdirectory of the `scripts` directory depending on the type. Tab scripts go in `scripts/tab`, URL scripts in `scripts/url`.

When you view actions for a Safari tab, both tab and URL actions are listed. When you action a bookmark, only URL actions are listed.

Tab scripts are called with the indices of the selected window and tab as `$1` and `$2`. So if the third tab of the second Safari window is active, your script is called as `/path/to/script 2 3`.

URL scripts are called with the URL of the selected bookmark or tab as `$1`, e.g. `/path/to/script http://www.example.com`.

See the built-in scripts (in the `scripts` subdirectory of the workflow) for examples of how to implement them.


<a name="supported-languages"></a>
### Supported languages

The workflow knows to run `.scpt`, `.js`, `.applescript` and `.scptd` scripts via `/usr/bin/osascript`. It can also run any script/program with its executable bit set (it will call these directly).


<a name="script-icons"></a>
### Script icons

By default, tab scripts get a tab icon and URL scripts a URL one. You can supply a custom icon for any script by saving the icon alongside the script with the same basename (i.e. the same name as the script, only with a different file extension). Supported icon extensions are `.png`, `.icns`, `.jpg`, `.jpeg` and `.gif`.


<a name="built-in-actions"></a>
### Built-in actions

The following actions are built into the workflow, either hard-coded or as bundled scripts (in the `scripts` subdirectory of the workflow).


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


<a name="history"></a>
History
-------

In Sierra and earlier, it was possible to search Safari's history with a File Filter. Unfortunately, the exported history files were removed in High Sierra, so it's now necessary to search Safari's SQLite History database instead.

The workflow accesses this database in two different ways.

The history search (keyword `hi`) uses SQLite's own search function, as there are too many entries to load and/or fuzzy search. As a result, the history search does *not* use fuzzy search.

The combined bookmark and recent history search (keyword `bh`) does use fuzzy search, but the trade-off is that it only reads a limited number of the most recent entries from the history database (specified by the `ALSF_HISTORY_ENTRIES` configuration option; 1000 by default).

Depending on the speed of your Mac and your own tolerance for slowness, you may be able to increase this number significantly.


<a name="licensing--thanks"></a>
Licensing & thanks
------------------

This workflow is released under the [MIT Licence][mit].

It is heavily based on the [Kingpin][kingpin] and [AwGo][awgo] libraries (both also [MIT][mit]).

The icons are from [Elusive Icons][elusive], [Font Awesome][awesome], [Material Icons][material] (all [SIL][sil]) and [Octicons][octicons] ([MIT][mit]), via the [workflow icon generator][icongen].


[kingpin]: https://github.com/alecthomas/kingpin
[awgo]: https://github.com/deanishe/awgo
[sil]: http://scripts.sil.org/cms/scripts/page.php?site_id=nrsi&id=OFL
[mit]: https://opensource.org/licenses/MIT
[elusive]: https://github.com/aristath/elusive-iconfont
[awesome]: http://fortawesome.github.io/Font-Awesome/
[material]: http://zavoloklom.github.io/material-design-iconic-font/
[octicons]: https://octicons.github.com/
[icongen]: http://icons.deanishe.net
