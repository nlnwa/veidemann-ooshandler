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
	"fmt"
	"github.com/kennygrant/sanitize"
	log "github.com/sirupsen/logrus"
	"github.com/steakknife/bloomfilter"
	"hash/fnv"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"strings"
)

const (
	maxElements = 500000
	probCollide = 0.00000001
)

type OosHandler struct {
	bf  *bloomfilter.Filter
	dir string
}

func NewOosHandler(dir string) *OosHandler {
	bf, err := bloomfilter.NewOptimal(maxElements, probCollide)
	if err != nil {
		panic(err)
	}
	h := &OosHandler{
		bf:  bf,
		dir: dir,
	}
	h.ImportExisting()
	return h
}

func (o *OosHandler) ImportExisting() {
	files, err := ioutil.ReadDir(o.dir)
	if err != nil {
		log.Fatal(err)
	}

	i := 0
	for _, f := range files {
		if f.IsDir() {
			continue
		}

		f, err := os.Open(path.Join(o.dir, f.Name()))
		if err != nil {
			log.Errorf("Could not open file '%v'", "seeds.txt")
		}
		defer f.Close()

		buf := bufio.NewReader(f)
		for {
			l, err := buf.ReadString('\n')
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Errorf("Error reading from file: %v", err)
				break
			}

			line := strings.Trim(l, "\n")
			if line == "" {
				continue
			}
			u, _ := o.parseUriAndGroup(line)
			o.bloomContains(u)
			i++
		}
	}
	log.Printf("Imported %v existing URIs", i)
}

func (o *OosHandler) Handle(uri string) (exists bool) {
	u, g := o.parseUriAndGroup(uri)

	exists = o.bloomContains(u)
	if exists {
		exists = o.isInFile(u, g)
	}

	if !exists {
		o.write(u, g)
	}
	return
}

func (o *OosHandler) parseUriAndGroup(uri string) (*url.URL, string) {
	u, err := url.Parse(uri)
	if err != nil {
		log.Warnf("Error parsing uri '%v': %v", uri, err)
	}

	parts := strings.Split(sanitize.Name(u.Hostname()), ".")
	var g string
	if len(parts) >= 2 {
		g = parts[len(parts)-1:][0] + "_" + parts[len(parts)-2:][0][:2]
	} else {
		g = u.Host
	}
	return u, g
}

func (o *OosHandler) bloomContains(uri *url.URL) bool {
	x := fnv.New64()
	x.Write([]byte(uri.Host))

	exists := o.bf.Contains(x) // could have false positives
	o.bf.Add(x)

	return exists
}

func (o *OosHandler) write(uri *url.URL, group string) {
	f, err := os.OpenFile(o.createFileName(group), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Warnf("Error opening file for writing: %v", err)
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%s://%s\n", uri.Scheme, uri.Host)
}

func (o *OosHandler) isInFile(uri *url.URL, group string) bool {
	f, err := os.Open(o.createFileName(group))
	if err != nil {
		log.Debugf("Error opening file: %v", err)
		return false
	}
	defer f.Close()
	buf := bufio.NewReader(f)
	for {
		l, err := buf.ReadString('\n')
		if err == nil {
			if l == fmt.Sprintf("%s://%s\n", uri.Scheme, uri.Host) {
				return true
			}
		} else {
			log.Debugf("Error checking file: %v", err)
			break
		}
	}
	return false
}

func (o *OosHandler) createFileName(group string) string {
	return path.Join(o.dir, "uri_"+group+".txt")
}
