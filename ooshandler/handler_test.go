/*
 * Copyright 2019 National Library of Norway.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *       http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package ooshandler

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOosHandler_Handle(t *testing.T) {
	testDir := "testoutput"

	wd, _ := os.Getwd()
	baseDir := filepath.Dir(wd)
	testDir = filepath.Join(baseDir, testDir)
	_ = os.Mkdir(testDir, 0777)
	defer os.RemoveAll(testDir)
	oos := NewOosHandler(testDir)

	u, _ := oos.parseUriAndGroup("http://example1.com")
	oos.bloomContains(u)

	type args struct {
		uri string
	}
	tests := []struct {
		name       string
		o          *OosHandler
		args       args
		wantExists bool
	}{
		{"1-1", oos, args{"http://example1.com"}, false},
		{"1-2", oos, args{"http://example2.com"}, false},
		{"1-3", oos, args{"http://example3.com"}, false},
		{"1-4", oos, args{"http://example2.com"}, true},
		{"1-5", oos, args{"http://example1.com"}, true},
		{"1-6", oos, args{"http://example3.com"}, true},
		{"2-1", oos, args{"http://example1.no"}, false},
		{"2-2", oos, args{"http://example2.no"}, false},
		{"3-1", oos, args{"http://bølle.no"}, false},
		{"3-2", oos, args{"http://bolle.no"}, false},
		{"3-3", oos, args{"https://bømållag.no"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := tt.o
			exists := o.Handle(tt.args.uri)
			if exists != tt.wantExists {
				t.Errorf("Expected %v, but got %v", tt.wantExists, exists)
			}
		})
	}

	fileValueTests := []struct {
		filename string
		values   []string
	}{
		{"uri_com_ex.txt", []string{"http://example1.com", "http://example2.com", "http://example3.com"}},
		{"uri_no_ex.txt", []string{"http://example1.no", "http://example2.no"}},
		{"uri_no_bo.txt", []string{"http://bølle.no", "http://bolle.no", "https://bømållag.no"}},
	}
	for _, ff := range fileValueTests {
		t.Run(ff.filename, func(t *testing.T) {
			f, err := os.Open(filepath.Join(testDir, ff.filename))
			if err != nil {
				t.Errorf("Could not open file '%v'", ff.filename)
			}
			defer f.Close()

			i := 0
			buf := bufio.NewReader(f)
			for {
				l, err := buf.ReadString('\n')
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Errorf("Error reading from file: %v", err)
					break
				}

				line := strings.Trim(l, "\n")
				if line == "" {
					break
				}
				if line != ff.values[i] {
					t.Errorf("Expected '%v', but got '%v'", ff.values[i], line)
				}
				i++
			}

			if i != len(ff.values) {
				t.Errorf("Expected %v values in file '%v', got %v", len(ff.values), ff.filename, i)
			}
		})
	}
}

func TestOosHandler_Import(t *testing.T) {
	indata := "testdata"
	preimport := filepath.Join(indata, "preimport")
	fileWithDuplicates := "seeds.txt"

	wd, _ := os.Getwd()
	baseDir := filepath.Dir(wd)
	indata = filepath.Join(baseDir, indata)
	preimport = filepath.Join(baseDir, preimport)
	fileWithDuplicates = filepath.Join(indata, fileWithDuplicates)

	oos := NewOosHandler(preimport)

	f, err := os.Open(fileWithDuplicates)
	if err != nil {
		t.Errorf("Could not open file '%v'", fileWithDuplicates)
	}
	defer f.Close()

	i := 0
	dup := 0
	buf := bufio.NewReader(f)
	c := make(chan bool)

	for {
		l, err := buf.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Errorf("Error reading from file: %v", err)
			break
		}

		line := strings.Trim(l, "\n")
		if line == "" {
			continue
		}

		i++
		go func() { c <- oos.Handle(line) }()
	}
	for j := 0; j < i; j++ {
		exists := <-c
		if exists {
			dup++
		}
	}
	if i != dup {
		t.Errorf("Expected all %v URIs to be pre imported, but found only %v", i, dup)
	}
}
