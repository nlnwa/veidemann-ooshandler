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
	"io"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/bits-and-blooms/bloom/v3"
	"github.com/kennygrant/sanitize"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/singleflight"
)

const (
	maxElements = 500000
	probCollide = 0.00000001
)

type OosHandler struct {
	bf          *bloom.BloomFilter
	dir         string
	bloomSF     singleflight.Group
	fmu         sync.Mutex // protects fileMutexes
	fileMutexes map[string]*sync.Mutex
}

func NewOosHandler(dir string) *OosHandler {
	bf := bloom.NewWithEstimates(maxElements, probCollide)
	h := &OosHandler{
		bf:          bf,
		dir:         dir,
		fileMutexes: make(map[string]*sync.Mutex),
	}
	h.ImportExisting()
	return h
}

func (o *OosHandler) ImportExisting() {
	files, err := os.ReadDir(o.dir)
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
			log.Errorf("Could not open file '%v'", path.Join(o.dir, f.Name()))
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
			u, _, err := o.parseUriAndGroup(line)
			if err != nil {
				log.Warn(err)
				continue
			}
			o.bloomContains(u)
			i++
		}
	}
	log.Printf("Imported %v existing URIs", i)
}

func (o *OosHandler) Handle(uri string) (exists bool) {
	u, g, err := o.parseUriAndGroup(uri)
	if err != nil {
		log.Warn(err)
		return false
	}

	v, _, _ := o.bloomSF.Do(u.Host, func() (interface{}, error) {
		exists = o.bloomContains(u)
		if exists {
			exists = o.isInFile(u, g)
		} else {
			o.write(u, g)
		}
		return exists, nil
	})

	return v.(bool)
}

func (o *OosHandler) parseUriAndGroup(uri string) (*url.URL, string, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, "", fmt.Errorf("error parsing uri '%v': %v", uri, err)
	}

	parts := strings.Split(sanitize.Name(u.Hostname()), ".")
	var g string
	if len(parts) >= 2 {
		// Get last part of host and maximum two chars from the next to last part of host
		// www.example.com => com_ex
		// 127.0.0.1       => 1_0
		l := Min(2, len(parts[len(parts)-2]))
		g = parts[len(parts)-1:][0] + "_" + parts[len(parts)-2:][0][:l]
	} else {
		g = u.Host
	}
	return u, g, nil
}

func Min(x, y int) int {
	if x > y {
		return y
	}
	return x
}

func (o *OosHandler) bloomContains(uri *url.URL) bool {
	return o.bf.TestOrAddString(uri.Host)
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
	c := o.getFileLock(group)
	defer c.Unlock()

	f, err := os.OpenFile(o.createFileName(group), os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		log.Debugf("Error opening file: %v", err)
		return false
	}
	defer f.Close()

	var exists bool
	buf := bufio.NewReader(f)
	for {
		l, err := buf.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				log.Debugf("Error checking file: %v", err)
			}
			break
		}
		l = strings.Trim(l, "\n")
		if l == fmt.Sprintf("%s://%s", uri.Scheme, uri.Host) {
			exists = true
			break
		}
	}

	if !exists {
		_, err := f.Seek(0, 2)
		if err != nil {
			log.Errorf("Error seeking to end of file: %v", err)
		} else {
			fmt.Fprintf(f, "%s://%s\n", uri.Scheme, uri.Host)
		}
	}

	return exists
}

func (o *OosHandler) createFileName(group string) string {
	return path.Join(o.dir, "uri_"+group+".txt")
}

func (o *OosHandler) getFileLock(key string) *sync.Mutex {
	o.fmu.Lock()
	defer o.fmu.Unlock()
	var c *sync.Mutex
	var ok bool
	if c, ok = o.fileMutexes[key]; ok {
		c.Lock()
	} else {
		c = new(sync.Mutex)
		c.Lock()
		o.fileMutexes[key] = c
	}

	return c
}
