// goscrutinize lints the Go source files named on its command line.
package main

import (
	"flag"
	"fmt"
	"go/build"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/golang/lint"
	"github.com/phayes/checkstyle"
)

var SDFSDFSDF = "SDFSDF"

const (
	outputFormatDefault    = "default"
	outputFormatCheckStyle = "checkstyle-xml"
)

var (
	minConfidence = flag.Float64("min_confidence", 0.8, "minimum confidence of a problem to print it")
	setExitStatus = flag.Bool("set_exit_status", false, "set exit status to 1 if any issues are found")
	outputFormat  = flag.String("output_format", "default", "set the output format. Allowed values are \"default\" and \"checkstyle-xml\"")
	suggestions   int
	cstyle        = checkstyle.New()
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "\tgolint [flags] # runs on package in current directory\n")
	fmt.Fprintf(os.Stderr, "\tgolint [flags] [packages]\n")
	fmt.Fprintf(os.Stderr, "\tgolint [flags] [directories] # where a '/...' suffix includes all sub-directories\n")
	fmt.Fprintf(os.Stderr, "\tgolint [flags] [files] # all must belong to a single package\n")
	fmt.Fprintf(os.Stderr, "Flags:\n")
	flag.PrintDefaults()
}

func main() {
	flag.Usage = usage
	flag.Parse()

	// Check output format
	if *outputFormat != outputFormatDefault && *outputFormat != outputFormatCheckStyle {
		fmt.Fprintf(os.Stderr, "Invalid output_format \"%v\". Allowed values are \"default\" and \"checkstyle-xml\"", *outputFormat)
		os.Exit(1)
	}

	if flag.NArg() == 0 {
		lintDir(".")
	} else {
		// dirsRun, filesRun, and pkgsRun indicate whether golint is applied to
		// directory, file or package targets. The distinction affects which
		// checks are run. It is no valid to mix target types.
		var dirsRun, filesRun, pkgsRun int
		var args []string
		for _, arg := range flag.Args() {
			if strings.HasSuffix(arg, "/...") && isDir(arg[:len(arg)-len("/...")]) {
				dirsRun = 1
				for _, dirname := range allPackagesInFS(arg) {
					args = append(args, dirname)
				}
			} else if isDir(arg) {
				dirsRun = 1
				args = append(args, arg)
			} else if exists(arg) {
				filesRun = 1
				args = append(args, arg)
			} else {
				pkgsRun = 1
				args = append(args, arg)
			}
		}

		if dirsRun+filesRun+pkgsRun != 1 {
			usage()
			os.Exit(2)
		}
		switch {
		case dirsRun == 1:
			for _, dir := range args {
				lintDir(dir)
			}
		case filesRun == 1:
			lintFiles(args...)
		case pkgsRun == 1:
			for _, pkg := range importPaths(args) {
				lintPackage(pkg)
			}
		}
	}

	if *outputFormat == outputFormatCheckStyle {
		fmt.Println(`<?xml version="1.0" encoding="UTF-8"?>`)
		fmt.Println(cstyle)
	}

	if *setExitStatus && suggestions > 0 {
		fmt.Fprintf(os.Stderr, "Found %d lint suggestions; failing.\n", suggestions)
		os.Exit(1)
	}
}

func isDir(filename string) bool {
	fi, err := os.Stat(filename)
	return err == nil && fi.IsDir()
}

func exists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

func lintFiles(filenames ...string) {
	files := make(map[string][]byte)
	for _, filename := range filenames {
		src, err := ioutil.ReadFile(filename)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		files[filename] = src
		cstyle.EnsureFile(filename)
	}

	l := new(lint.Linter)
	ps, err := l.LintFiles(files)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return
	}
	for _, p := range ps {
		if p.Confidence >= *minConfidence {
			if *outputFormat == outputFormatDefault {
				fmt.Printf("%v: %s\n", p.Position, p.Text)
			} else if *outputFormat == outputFormatCheckStyle {
				checkstyleFile := cstyle.EnsureFile(p.Position.Filename)
				checkstyleError := checkstyle.NewError(p.Position.Line, p.Position.Column, checkstyle.SeverityInfo, p.Text, p.Category)
				checkstyleFile.AddError(checkstyleError)
			}
			suggestions++
		}
	}
}

func lintDir(dirname string) {
	pkg, err := build.ImportDir(dirname, 0)
	lintImportedPackage(pkg, err)
}

func lintPackage(pkgname string) {
	pkg, err := build.Import(pkgname, ".", 0)
	lintImportedPackage(pkg, err)
}

func lintImportedPackage(pkg *build.Package, err error) {
	if err != nil {
		if _, nogo := err.(*build.NoGoError); nogo {
			// Don't complain if the failure is due to no Go source files.
			return
		}
		fmt.Fprintln(os.Stderr, err)
		return
	}

	var files []string
	files = append(files, pkg.GoFiles...)
	files = append(files, pkg.CgoFiles...)
	files = append(files, pkg.TestGoFiles...)
	if pkg.Dir != "." {
		for i, f := range files {
			files[i] = filepath.Join(pkg.Dir, f)
		}
	}
	// TODO(dsymonds): Do foo_test too (pkg.XTestGoFiles)

	lintFiles(files...)
}
