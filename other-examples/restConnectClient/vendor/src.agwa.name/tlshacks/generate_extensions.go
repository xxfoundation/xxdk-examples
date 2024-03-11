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

//go:generate go run generate_extensions.go

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

const sourceURL = `https://www.iana.org/assignments/tls-extensiontype-values/tls-extensiontype-values-1.csv`

const outputFilename = "extensions.go"

const outputFormat = `// GENERATED BY generate_extensions.go - DO NOT EDIT

package tlshacks

var Extensions = %#v
`

type ExtensionInfo = struct {
	Name      string
	Reserved  bool
	Grease    bool
	Private   bool
	Reference string
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

	extensions := make(map[uint16]ExtensionInfo)

	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			log.Fatal(err)
		}

		var (
			valueString = row[0]
			name        = row[1]
			reference   = row[5]
		)
		if name == "Unassigned" {
			continue
		}
		if index := strings.Index(name, " (renamed from"); index != -1 {
			name = name[:index]
		}
		var (
			firstValue uint64
			lastValue  uint64
		)
		if valueFields := strings.SplitN(valueString, "-", 2); len(valueFields) == 2 {
			firstValue, err = strconv.ParseUint(valueFields[0], 0, 16)
			if err != nil {
				log.Fatal(err)
			}
			lastValue, err = strconv.ParseUint(valueFields[1], 0, 16)
			if err != nil {
				log.Fatal(err)
			}
		} else {
			firstValue, err = strconv.ParseUint(valueString, 0, 16)
			if err != nil {
				log.Fatal(err)
			}
			lastValue = firstValue
		}

		var info ExtensionInfo
		if name == "Reserved" && reference == "[RFC8701]" {
			info.Reserved = true
			info.Grease = true
		} else if name == "Reserved for Private Use" {
			info.Reserved = true
			info.Private = true
		} else if name == "Reserved" {
			info.Reserved = true
		} else {
			info.Name = name
		}
		info.Reference = reference
		for value := firstValue; value <= lastValue; value++ {
			extensions[uint16(value)] = info
		}
	}

	if err := os.WriteFile(outputFilename, []byte(fmt.Sprintf(outputFormat, extensions)), 0666); err != nil {
		log.Fatal(err)
	}
}