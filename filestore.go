/*
 *  filestore - Filestore API.
 *
 *  Copyright (c) 2019  Priceboro Newport, Inc.  All Rights Reserved.
 *
 */

package filestore

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

type FileStore struct {
	name          string
	lock          sync.RWMutex
	last_modified time.Time
	values        map[string]string
	comments      []string
}

func LastModified(name string) time.Time {
	fi, _ := os.Stat(name)
	if fi == nil {
		return time.Time{}
	}
	return fi.ModTime()
}

func (fs *FileStore) load_values() {
	f, err := os.Open(fs.name)
	if err != nil {
		return
	}
	defer f.Close()
	for k := range fs.values {
		delete(fs.values, k)
	}
	fs.comments = nil
	s := bufio.NewScanner(f)
	for s.Scan() {
		rec := s.Text()
		if len(rec) > 0 && rec[0:1] == "#" {
			fs.comments = append(fs.comments, rec)
		} else {
			i := strings.Index(rec, "=")
			n := rec
			v := ""
			if i >= 0 {
				n = rec[:i]
				v = rec[i+1:]
			}
			fs.values[n] = v
		}
	}
	fs.last_modified = LastModified(fs.name)
}

func New(name string) *FileStore {
	last_modified := time.Now()
	f, _ := os.Open(name)
	if f == nil {
		f, _ = os.Create(name)
	} else {
		last_modified = LastModified(name)
	}
	if f == nil {
		return nil
	}
	fs := FileStore{name: name, last_modified: last_modified, values: make(map[string]string), comments: make([]string, 10)}
	fs.load_values()
	return &fs
}

func (fs *FileStore) Read(keys ...string) (result string) {
	if len(keys) > 0 {
		fs.lock.RLock()
		last_modified := LastModified(fs.name)
		if last_modified.After(fs.last_modified) {
			fs.load_values()
		}
		result = fs.values[keys[0]]
		if result == "" && len(keys) > 1 {
			result = keys[1]
		}
		fs.lock.RUnlock()
	}
	return
}

func (fs *FileStore) store_values() (err error) {
	f, err := os.Create(fs.name)
	if err == nil {
		defer f.Close()
		for i := range fs.comments {
			fmt.Fprintf(f, "%s\n", fs.comments[i])
		}
		for n, v := range fs.values {
			fmt.Fprintf(f, "%s=%s\n", n, v)
		}
		fs.last_modified = time.Now()
	}
	return
}

func (fs *FileStore) Write(key string, value string) (err error) {
	fs.lock.Lock()
	if fs.values[key] != value {
		fs.values[key] = value
		err = fs.store_values()
	}
	fs.lock.Unlock()
	return
}
