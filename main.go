package main

import (
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/phayes/checkstyle"
)

var (
	regexGithub    = regexp.MustCompile("^g")
	regexBitbucket = regexp.MustCompile("^b")
	regexGitURI    = regexp.MustCompile("((git|ssh|http(s)?)|(git@[\\w\\.]+))(:(//)?)([\\w\\.@\\:/\\-~]+)(\\.git)(/)?")
)

var (
	cstyle                                                   = checkstyle.New()
	homeEnv                                                  = os.Getenv("HOME")
	gopathEnv                                                = os.Getenv("GOPATH")
	scrutinizerProjectEnv                                    = os.Getenv("SCRUTINIZER_PROJECT")
	projectFull, projectDomain, projectOwner, projectProject string
)

func main() {
	// Set up environment variables
	if gopathEnv == "" {
		gopathEnv = homeEnv + "/go"
	}

	// Set up project
	if len(scrutinizerProjectEnv) == 0 {
		log.Fatal("Not running without scrutinizer environemnt. SCRUTINIZER_PROJECT environment variable not found")
	}

	projectFull = regexGithub.ReplaceAllString(scrutinizerProjectEnv, "github.com")
	projectFull = regexBitbucket.ReplaceAllString(projectFull, "bitbucket.com")

	projectParts := strings.Split(projectFull, "/")
	if len(projectParts) != 3 {
		log.Fatal("Malformed SCRUTINIZER_PROJECT environment variable.")
	}
	projectDomain = projectParts[0]
	projectOwner = projectParts[1]
	projectProject = projectParts[2]

	// Install all dependancies
	cmd := exec.Command("go", "get", "-t", "./...")
	_, err := cmd.Output()
	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if ok && len(exitErr.Stderr) != 0 {
			log.Println(string(exitErr.Stderr))
		}
		log.Fatal("go get -t ./...", err)
	}

	// Run metalinter
	metalinter()

	// Run tests and coverage
	testAndCoverage()
}

func metalinter() {
	goMetaLinterCmd := gopathEnv + "/bin/gometalinter"

	// Install gometalinter -- no-op if already installed
	cmd := exec.Command("go", "get", "github.com/alecthomas/gometalinter")
	_, err := cmd.Output()
	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if ok && len(exitErr.Stderr) != 0 {
			log.Println(string(exitErr.Stderr))
		}
		log.Fatal("go get github.com/alecthomas/gometalinter", err)
	}

	// Install all gometalinter dependancies -- no-op if already installed
	cmd = exec.Command(goMetaLinterCmd, "--install")
	_, err = cmd.Output()
	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if ok && len(exitErr.Stderr) != 0 {
			log.Println(string(exitErr.Stderr))
		}
		log.Fatal(goMetaLinterCmd, "--install", err)
	}

	// Configure the metalinter
	if _, err := os.Stat("go-scrutinize.config"); os.IsNotExist(err) {
		cmd = exec.Command(goMetaLinterCmd, "--checkstyle")
	} else {
		cmd = exec.Command(goMetaLinterCmd, "--checkstyle", "--config=go-scrutinize.config")
	}

	// Run the metalinter
	out, err := cmd.Output()
	if err != nil {
		exitErr := err.(*exec.ExitError)
		if len(exitErr.Stderr) != 0 {
			log.Println(string(exitErr.Stderr))
		}
		log.Println(string(out))
		log.Fatal(goMetaLinterCmd, err)
	}

	// Write the output from the metalinter
	ioutil.WriteFile("checkstyle_report.xml", out, os.ModePerm)
}

func testAndCoverage() {
	goConvCmd := gopathEnv + "/bin/gocov"
	goConvXMLCmd := gopathEnv + "/bin/gocov-xml"

	// Install the coverage tool covov
	cmd := exec.Command("go", "get", "github.com/axw/gocov/...")
	_, err := cmd.Output()
	if err != nil {
		exitErr := err.(*exec.ExitError)
		if len(exitErr.Stderr) != 0 {
			log.Println(string(exitErr.Stderr))
		}
		log.Fatal("go", "get", "github.com/axw/gocov/...", err)
	}

	// Install the coverage file translation tool gocov-xml
	cmd = exec.Command("go", "get", "github.com/AlekSi/gocov-xml")
	_, err = cmd.Output()
	if err != nil {
		exitErr := err.(*exec.ExitError)
		if len(exitErr.Stderr) != 0 {
			log.Println(string(exitErr.Stderr))
		}
		log.Fatal("go", "get", "github.com/AlekSi/gocov-xml", err)
	}

	// Run tests with coverage
	cmd = exec.Command(goConvCmd, "test", "./...", "-race")
	gocovout, err := cmd.Output()
	if err != nil {
		exitErr := err.(*exec.ExitError)
		if len(exitErr.Stderr) != 0 {
			log.Println(string(exitErr.Stderr))
		}
		log.Fatal(goConvCmd, "test", "./...", "-race", err)
	}

	// Convert to clover format
	cmd = exec.Command(goConvXMLCmd)
	cmd.Stdin = bytes.NewReader(gocovout)
	xmlout, err := cmd.Output()
	if err != nil {
		exitErr := err.(*exec.ExitError)
		if len(exitErr.Stderr) != 0 {
			log.Println(string(exitErr.Stderr))
		}
		log.Fatal(goConvXMLCmd, err)
	}

	// Rewrite all filenames to use /home/scrutinizer/build paths
	coveragexml := strings.Replace(string(xmlout), gopathEnv+"/src/"+projectFull, "/home/scrutinizer/build", 0)

	// Write the output from the metalinter
	ioutil.WriteFile("coverage.xml", []byte(coveragexml), os.ModePerm)

}
