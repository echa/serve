// Copyright (c) 2019 KIDTSUNAMI
// Author: alex@kidtsunami.com

package server

import (
	"github.com/echa/log"
	"mime"
)

var (
	mimeTypes map[string]string = map[string]string{
		".atom":  "application/atom+xml",
		".css":   "text/css",
		".csv":   "text/csv",
		".eot":   "application/vnd.ms-fontobject",
		".gif":   "image/gif",
		".ico":   "image/x-icon",
		".jar":   "application/java-archive",
		".jp2":   "image/jp2",
		".jpeg":  "image/jpeg",
		".jpg":   "image/jpeg",
		".js":    "application/javascript",
		".json":  "application/json",
		".otf":   "application/font-sfnt",
		".pdf":   "application/pdf",
		".png":   "image/png",
		".ps":    "application/postscript",
		".rss":   "application/rss+xml",
		".rtf":   "application/rtf",
		".svg":   "image/svg+xml",
		".tiff":  "image/tiff",
		".ttf":   "application/font-sfnt",
		".webp":  "image/webp",
		".woff":  "application/font-woff",
		".woff2": "font/woff2",
		".xml":   "application/xml",
		".zip":   "application/zip",
	}
)

func init() {
	for ext, typ := range mimeTypes {
		if err := mime.AddExtensionType(ext, typ); err != nil {
			log.Fatalf("mime type init failed: %v", err)
		}
	}
}
