// Copyright (c) 2019 Dean Jackson <deanishe@deanishe.net>
// MIT Licence applies http://opensource.org/licenses/MIT

// +build mage

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"

	"github.com/deanishe/awgo/util/build"
)

// Default target to run when none is specified
// If not set, running mage will list available targets
// var Default = Build

var (
	info     *build.Info
	buildDir = "./build"
	distDir  = "./dist"
)

func init() {
	var err error
	if info, err = build.NewInfo(); err != nil {
		panic(err)
	}
}

func mod(args ...string) error {
	argv := append([]string{"mod"}, args...)
	return sh.RunWith(info.Env(), "go", argv...)
}

// Aliases are mage command aliases.
var Aliases = map[string]interface{}{
	"b": Build,
	"c": Clean,
	"d": Dist,
	"l": Link,
}

// Build builds workflow in ./build
func Build() error {
	mg.Deps(cleanBuild, Icons)

	if err := sh.RunWith(info.Env(),
		"go", "build", "-o", "./build/alsf", ".",
	); err != nil {
		return err
	}

	globs := build.Globs(
		"*.png",
		"info.plist",
		"README.*",
		"LICENCE.txt",
		"icons/*.png",
		"scripts/tab/*",
		"scripts/url/*",
	)

	return build.SymlinkGlobs(buildDir, globs...)
}

// Run run workflow
func Run() error {
	mg.Deps(Build)
	fmt.Println("running ...")
	return sh.RunWith(info.Env(), "./build/alsf", "-h")
}

// Dist build an .alfredworkflow file in ./dist
func Dist() error {
	mg.SerialDeps(Build)
	p, err := build.Export(buildDir, distDir)
	if err != nil {
		return err
	}
	fmt.Printf("exported %q\n", p)
	return nil
}

// Link symlinks ./build directory to Alfred's workflow directory.
func Link() error {
	mg.Deps(Build)
	dir := filepath.Join(info.AlfredWorkflowDir, info.BundleID)
	fmt.Printf("symlinking %q to %q ...\n", buildDir, dir)
	return build.Symlink(dir, buildDir, true)
}

// Icons generate icons
func Icons() error {
	var (
		green  = "00e756"
		yellow = "f8ac30"
		// blue  = "00a1de"
		// red   = "c92441"
	)

	copies := []struct {
		src, dest, colour string
	}{
		{"docs.png", "help.png", green},
		{"tab.png", "tab-active.png", yellow},
	}

	for _, cfg := range copies {
		var (
			src  = filepath.Join("icons", cfg.src)
			dest = filepath.Join("icons", cfg.dest)
		)
		if err := copyImage(src, dest, cfg.colour); err != nil {
			return err
		}
	}

	return nil
}

// Deps ensure dependencies
func Deps() error {
	mg.Deps(cleanDeps)
	fmt.Println("downloading deps ...")
	return mod("download")
}

// Clean remove build files
func Clean() {
	mg.Deps(cleanDeps, cleanBuild, cleanMage)
}

func cleanDeps() error {
	fmt.Println("tidying deps ...")
	return mod("tidy", "-v")
}

func cleanBuild() error {
	fmt.Printf("cleaning %s ...\n", buildDir)
	return cleanDir(buildDir)
}

func cleanMage() error {
	fmt.Println("cleaning mage cache ...")
	return sh.Run("mage", "-clean")
}

// CleanIcons delete all generated icons from ./icons
func CleanIcons() error {
	return cleanDir("./icons", "safari.png")
}

func cleanDir(name string, exclude ...string) error {
	if _, err := os.Stat(name); err != nil {
		return nil
	}

	infos, err := ioutil.ReadDir(name)
	if err != nil {
		return err
	}
	for _, fi := range infos {

		var match bool
		for _, glob := range exclude {
			if match, err = doublestar.Match(glob, fi.Name()); err != nil {
				return err
			} else if match {
				break
			}
		}

		if match {
			fmt.Printf("excluded: %s\n", fi.Name())
			continue
		}

		p := filepath.Join(name, fi.Name())
		if err := os.RemoveAll(p); err != nil {
			return err
		}
	}
	return nil
}

// expand ~ and variables in path.
func expandPath(path string) string {
	if strings.HasPrefix(path, "~") {
		path = "${HOME}" + path[1:]
	}

	return os.ExpandEnv(path)
}

func exists(path string) bool {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false
		}
		panic(err)
	}

	return true
}
