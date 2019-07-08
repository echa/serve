// Copyright (c) 2019 KIDTSUNAMI
// Author: alex@kidtsunami.com

package server

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Du to the use of bytes.Reader max file size will be limited to 2^32
// and practically to a much lower user defined value. This helps prevent
// heap memory exhaustion.
var MaxFileSize int64 = 1 << 24 // 16 MB

type CachedFile struct {
	buf []byte
	rd  *bytes.Reader
	fi  *CachedFileInfo
}

type CachedFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modtime time.Time
}

func NewCachedFileInfo(fi os.FileInfo) *CachedFileInfo {
	return &CachedFileInfo{
		name:    fi.Name(),
		size:    fi.Size(),
		mode:    fi.Mode(),
		modtime: fi.ModTime(),
	}
}

func (i *CachedFileInfo) Name() string       { return i.name }
func (i *CachedFileInfo) Size() int64        { return i.size }
func (i *CachedFileInfo) Mode() os.FileMode  { return i.mode }
func (i *CachedFileInfo) ModTime() time.Time { return i.modtime }
func (i *CachedFileInfo) IsDir() bool        { return false }
func (i *CachedFileInfo) Sys() interface{}   { return nil }

func NewCachedFile(f http.File) (*CachedFile, error) {
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if fi.Size() > MaxFileSize {
		return nil, io.ErrShortBuffer
	}
	buf, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	return &CachedFile{buf: buf, rd: bytes.NewReader(buf), fi: NewCachedFileInfo(fi)}, nil
}

func IsCached(f http.File) bool {
	_, ok := f.(*CachedFile)
	return ok
}

func (f *CachedFile) Read(p []byte) (n int, err error) {
	return f.rd.Read(p)
}

func (f *CachedFile) Seek(offset int64, whence int) (int64, error) {
	return f.rd.Seek(offset, whence)
}

func (f *CachedFile) Close() error {
	f.rd.Seek(0, io.SeekStart)
	return nil
}

func (f *CachedFile) Readdir(count int) ([]os.FileInfo, error) {
	return nil, os.ErrInvalid
}

func (f *CachedFile) Stat() (os.FileInfo, error) {
	return f.fi, nil
}

func (f *CachedFile) ReplaceTemplates() {
	buf := bytes.NewBuffer(make([]byte, 0, len(f.buf)))
	FindAndReplace(f.buf, buf, func(v string) string {
		return os.Getenv(v)
	})
	f.buf = buf.Bytes()
	f.rd = bytes.NewReader(f.buf)
	f.fi.size = int64(len(f.buf))
}

func CheckDir(path string) error {
	if path == "" {
		path = "."
	}
	d, err := os.Stat(path)
	if os.IsNotExist(err) {
		return fmt.Errorf("directory '%s' not found", path)
	}
	if !d.Mode().IsDir() {
		return fmt.Errorf("'%s' is not a directory", path)
	}
	if (d.Mode().Perm() & 0444) == 0 {
		return fmt.Errorf("'%s' is not readable", path)
	}
	return nil
}

func CheckFile(path, name string) error {
	path = filepath.Join(path, name)
	if path == "" {
		return fmt.Errorf("missing file name")
	}
	if err := CheckDir(filepath.Dir(path)); err != nil {
		return err
	}
	d, err := os.Stat(path)
	if os.IsNotExist(err) {
		return fmt.Errorf("file '%s' not found", path)
	}
	if d.Mode().IsDir() {
		return fmt.Errorf("'%s' is a directory", path)
	}
	if (d.Mode().Perm() & 0444) == 0 {
		return fmt.Errorf("'%s' is not readable", path)
	}
	return nil
}
