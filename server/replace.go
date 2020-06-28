// Copyright (c) 2019 KIDTSUNAMI
// Author: alex@kidtsunami.com

package server

import (
	"bytes"
	"io"
)

var (
	startDelim = []byte("[[")
	endDelim   = []byte("]]")
	maxReplace = 32
)

func SetDelims(start, end string) {
	startDelim = []byte(start)
	endDelim = []byte(end)
}

func SetMaxReplace(size int) {
	maxReplace = size
}

func FindTemplates(buf []byte) (locs []int) {
	locs = make([]int, 0)
	var (
		found  = -1
		start  int
		end    int
		length = len(buf)
	)

	for i := 0; i < length; i++ {
		found = bytes.Index(buf[i:], startDelim)
		if -1 == found {
			break
		}
		start = found + i

		found = bytes.Index(buf[start+len(startDelim):], endDelim)
		if -1 == found {
			break
		}
		end = found + start + len(endDelim)

		if end-start > maxReplace {
			continue
		}

		locs = append(locs, start, end)
		i = end
	}
	return
}

func ReplaceTemplates(src []byte, w io.Writer, locs []int, fn func(string) string) {
	var (
		next   int
		last   int
		length = len(locs)
	)
	for i := 0; i < length; i += 2 {
		next = locs[i]
		w.Write(src[last:next])
		rep := fn(string(src[next+len(startDelim) : locs[i+1]]))
		if len(rep) > 0 {
			w.Write([]byte(rep))
		}
		last = locs[i+1] + len(endDelim)
	}
	w.Write(src[last:])
}

func FindAndReplace(buf []byte, w io.Writer, fn func(string) string) {
	ReplaceTemplates(buf, w, FindTemplates(buf), fn)
}
