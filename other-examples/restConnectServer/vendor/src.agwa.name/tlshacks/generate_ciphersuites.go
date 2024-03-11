// Copyright (C) 2022 Andrew Ayer
//
// Permission is hereby granted, free of charge, to any person obtaining a
// copy of this software and associated documentation files (the "Software"),
// to deal in the Software without restriction, including without limitation
// the rights to use, copy, modify, merge, publish, distribute, sublicense,
// and/or sell copies of the Software, and to permit persons to whom the
// Software is furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included
// in all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL
// THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR
// OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE,
// ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR
// OTHER DEALINGS IN THE SOFTWARE.
//
// Except as contained in this notice, the name(s) of the above copyright
// holders shall not be used in advertising or otherwise to promote the
// sale, use or other dealings in this Software without prior written
// authorization.

//go:build generate

//go:generate go run generate_ciphersuites.go

package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const sourceURL = `https://www.iana.org/assignments/tls-parameters/tls-parameters-4.csv`

const outputFilename = "ciphersuites.go"

const outputFormat = `// GENERATED BY generate_ciphersuites.go - DO NOT EDIT

package tlshacks

var CipherSuites = %#v
`

type CipherSuiteInfo = struct {
	Name   string
	Grease bool
}

func main() {
	client := &http.Client{Timeout: 1 * time.Minute}
	resp, err := client.Get(sourceURL)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Fatalf("%s: %d %s", sourceURL, resp.StatusCode, resp.Status)
	}

	reader := csv.NewReader(resp.Body)

	// Discard header
	if _, err := reader.Read(); err != nil {
		log.Fatal(err)
	}

	ciphersuites := make(map[uint16]CipherSuiteInfo)

	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			log.Fatal(err)
		}

		var (
			value = row[0]
			desc  = row[1]
			ref   = row[4]
		)
		if desc == "Unassigned" {
			continue
		}
		if strings.Contains(desc, "Reserved") && ref != "[RFC8701]" {
			continue
		}
		valueFields := strings.Split(value, ",")
		hi, err := strconv.ParseUint(valueFields[0], 0, 8)
		if err != nil {
			log.Fatal(err)
		}
		lo, err := strconv.ParseUint(valueFields[1], 0, 8)
		if err != nil {
			log.Fatal(err)
		}

		code := (uint16(hi) << 8) | uint16(lo)
		if strings.Contains(desc, "Reserved") && ref == "[RFC8701]" {
			ciphersuites[code] = CipherSuiteInfo{Grease: true}
		} else {
			ciphersuites[code] = CipherSuiteInfo{Name: desc}
		}
	}

	if err := os.WriteFile(outputFilename, []byte(fmt.Sprintf(outputFormat, ciphersuites)), 0666); err != nil {
		log.Fatal(err)
	}
}