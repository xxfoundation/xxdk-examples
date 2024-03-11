////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

// Provides functions for writing a version information file

package utils

import (
	"bufio"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"text/template"
	"time"
)

// Version file generation consumed by higher-level repos
func GenerateVersionFile(version string) {
	gitversion := GenerateGitVersion()
	deps := ReadGoMod()

	f, err := os.Create("version_vars.go")
	if err != nil {
		panic(err)
	}

	err = packageTemplate.Execute(f, struct {
		Timestamp    time.Time
		GITVER       string
		DEPENDENCIES string
		VERSION      string
	}{
		Timestamp:    time.Now(),
		GITVER:       gitversion,
		DEPENDENCIES: deps,
		VERSION:      version,
	})
	if err != nil {
		panic(err)
	}

	err = f.Close()
	if err != nil {
		panic(err)
	}
}

// Determine current Git version information
func GenerateGitVersion() string {
	cmd := exec.Command("git", "show", "--oneline")
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		panic(err)
	}
	scanner := bufio.NewScanner(strings.NewReader(string(stdoutStderr)))
	for scanner.Scan() {
		return scanner.Text()
	}
	return "UNKNOWNVERSION"
}

// Read in go modules file
func ReadGoMod() string {
	r, err := ioutil.ReadFile("go.mod")
	if err != nil {
		panic(err)
	}
	return string(r)
}

// Template for version_vars.go
var packageTemplate = template.Must(template.New("").Parse(
	"// Code generated by go generate; DO NOT EDIT.\n" +
		"// This file was generated by robots at\n" +
		"// {{ .Timestamp }}\n\n" +
		"package cmd\n\n" +
		"const GITVERSION = `{{ .GITVER }}`\n" +
		"const SEMVER = \"{{ .VERSION }}\"\n" +
		"const DEPENDENCIES = `{{ .DEPENDENCIES }}`\n"))
