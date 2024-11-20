// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

//go:build ignore

package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"text/template"
)

var (
	output = flag.String("output", "defs_pdh_windows.go", "output file")
)

const includes = `
#include <pdhmsg.h>
`

type TemplateParams struct {
	Errors []string
}

const fileTemplate = `
// go run mkpdh_defs.go
// MACHINE GENERATED BY THE ABOVE COMMAND; DO NOT EDIT

// +build ignore

package pdh

/*
#include <pdh.h>
#include <pdhmsg.h>
#cgo LDFLAGS: -lpdh
*/
import "C"

type PdhErrno uintptr

// PDH Error Codes
const (
{{- range $i, $errorCode := .Errors }}
	{{ $errorCode }} PdhErrno = C.{{ $errorCode }}
{{- end }}
)

var pdhErrors = map[PdhErrno]struct{}{
{{- range $i, $errorCode := .Errors }}
	{{ $errorCode }}: struct{}{},
{{- end }}
}

type PdhCounterFormat uint32

// PDH Counter Formats
const (
	// PdhFmtDouble returns data as a double-precision floating point real.
	PdhFmtDouble PdhCounterFormat = C.PDH_FMT_DOUBLE
	// PdhFmtLarge returns data as a 64-bit integer.
	PdhFmtLarge PdhCounterFormat = C.PDH_FMT_LARGE
	// PdhFmtLong returns data as a long integer.
	PdhFmtLong PdhCounterFormat = C.PDH_FMT_LONG

	// Use bitwise operators to combine these values with the counter type to scale the value.

    // Do not apply the counter's default scaling factor.
	PdhFmtNoScale PdhCounterFormat = C.PDH_FMT_NOSCALE
	// Counter values greater than 100 (for example, counter values measuring
	// the processor load on multiprocessor computers) will not be reset to 100.
	// The default behavior is that counter values are capped at a value of 100.
	PdhFmtNoCap100 PdhCounterFormat = C.PDH_FMT_NOCAP100
	// Multiply the actual value by 1,000.
	PdhFmtMultiply1000 PdhCounterFormat = C.PDH_FMT_1000
)
`

var (
	tmpl = template.Must(template.New("defs_pdh_windows").Parse(fileTemplate))

	pdhErrorRegex = regexp.MustCompile(`^#define (PDH_[\w_]+)`)
)

func main() {
	errors, err := getErrorDefines()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	t := TemplateParams{
		Errors: errors,
	}

	if err := writeOutput(t); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	if err := gofmtOutput(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func getErrorDefines() ([]string, error) {
	cmd := exec.Command("gcc", "-E", "-dD", "-")
	cmd.Stdin = bytes.NewBuffer([]byte(includes))
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var errors []string
	s := bufio.NewScanner(bytes.NewBuffer(out))
	for s.Scan() {
		line := s.Text()
		matches := pdhErrorRegex.FindStringSubmatch(line)
		if len(matches) > 1 {
			errors = append(errors, matches[1])
		}
	}

	return errors, nil
}

func writeOutput(p TemplateParams) error {
	// Create output file.
	f, err := os.Create(*output)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := tmpl.Execute(f, p); err != nil {
		return err
	}
	return nil
}

func gofmtOutput() error {
	_, err := exec.Command("gofmt", "-w", *output).Output()
	return err
}