// Copyright (c) 2019 KIDTSUNAMI
// Author: alex@kidtsunami.com

package server

import (
	"crypto/sha1"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/echa/config"
	"github.com/echa/log"
)

var idStream chan string

func init() {
	config.SetDefault("server.scheme", "http")
	config.SetDefault("server.addr", "0.0.0.0")
	config.SetDefault("server.port", 8000)
	config.SetDefault("server.root", ".")
	config.SetDefault("server.index", "index.html")
	config.SetDefault("template.enable", true)
	config.SetDefault("template.left", "<[") // may use {{}}, [[]], <%%> <##>, <<>>
	config.SetDefault("template.right", "]>")
	config.SetDefault("template.maxreplace", 32)
	config.SetDefault("template.maxsize", int64(16*1024*1024))
	config.SetDefault("cache.enable", true)
	config.SetDefault("cache.expires", 30*time.Second)
	config.SetDefault("cache.control", "public")

	// start async ID generator
	idStream = make(chan string, 100)
	go func(ch chan string) {
		h := sha1.New()
		c, _ := time.Now().MarshalBinary()
		for {
			h.Write(c)
			ch <- hex.EncodeToString(h.Sum(nil))
		}
	}(idStream)
}

type ServerConfig struct {
	Addr   string
	Port   int
	Scheme string
	Host   string
	Root   string
	Base   string
	Index  string
	CspLog string
	Cache  CacheConfig
	Tpl    TemplateConfig
}

type CacheConfig struct {
	Enable  bool
	Expires time.Duration
	Control string
	Rules   []CacheRule
}

type CacheRule struct {
	Filename string
	Regexp   *regexp.Regexp
	Ignore   bool
	NoCache  bool
	Expires  time.Duration
	Control  string
}

type TemplateConfig struct {
	Enable     bool
	Match      *regexp.Regexp
	Left       string
	Right      string
	MaxSize    int64
	MaxReplace int
}

type SPAServer struct {
	cfg     ServerConfig
	headers map[string]string
	root    http.FileSystem
	cache   map[string]http.File
}

func NewSPAServer() (*SPAServer, error) {
	srv := &SPAServer{
		cfg: ServerConfig{
			Addr:   config.GetString("server.addr"),
			Port:   config.GetInt("server.port"),
			Scheme: config.GetString("server.scheme"),
			Host:   config.GetString("server.host"),
			Root:   config.GetString("server.root"),
			Index:  config.GetString("server.index"),
			Base:   config.GetString("server.base"),
			CspLog: config.GetString("server.csplog"),
			Cache: CacheConfig{
				Enable:  config.GetBool("cache.enable"),
				Expires: config.GetDuration("cache.expires"),
				Control: config.GetString("cache.control"),
			},
			Tpl: TemplateConfig{
				Enable:     config.GetBool("template.enable"),
				Left:       config.GetString("template.left"),
				Right:      config.GetString("template.right"),
				MaxSize:    config.GetInt64("template.maxsize"),
				MaxReplace: config.GetInt("template.maxreplace"),
			},
		},
		headers: config.GetStringMap("headers"),
		root:    http.Dir(config.GetString("server.root")),
		cache:   make(map[string]http.File),
	}

	// set max filesize limit
	MaxFileSize = srv.cfg.Tpl.MaxSize

	// make sure server root exists and is readable
	if err := CheckDir(srv.cfg.Root); err != nil {
		return nil, fmt.Errorf("server root %v", err)
	}

	// make sure index file exists and is readable
	if err := CheckFile(srv.cfg.Root, srv.cfg.Index); err != nil {
		return nil, fmt.Errorf("server index %v", err)
	}

	// parse template matching config
	if restr := config.GetString("template.match"); len(restr) > 0 {
		re, err := regexp.Compile(restr)
		if err != nil {
			return nil, fmt.Errorf("parsing 'template.match' regexp: %v", err)
		}
		srv.cfg.Tpl.Match = re
		SetDelims(srv.cfg.Tpl.Left, srv.cfg.Tpl.Right)
		SetMaxReplace(srv.cfg.Tpl.MaxReplace)
	}

	// parse cache config rules
	err := config.ForEach("cache.rules", func(c *config.Config) error {
		rule := CacheRule{
			Filename: c.GetString("filename"),
			Ignore:   c.GetBool("ignore"),
			NoCache:  c.GetBool("nocache"),
			Expires:  c.GetDuration("expires"),
			Control:  c.GetString("control"),
		}
		if restr := c.GetString("regexp"); len(restr) > 0 {
			re, err := regexp.Compile(restr)
			if err != nil {
				return err
			}
			rule.Regexp = re
		}
		srv.cfg.Cache.Rules = append(srv.cfg.Cache.Rules, rule)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("cannot read cache config: %v", err)
	}
	return srv, nil
}

func (s *SPAServer) Address() string {
	return net.JoinHostPort(s.cfg.Addr, strconv.Itoa(s.cfg.Port))
}

func (s *SPAServer) TLS() *tls.Config {
	if s.cfg.Scheme != "https" {
		return nil
	}
	tlsc, err := NewTLSConfig(TLSConfig{
		ServerName:      config.GetString("server.name"),
		TLSMinVersion:   config.GetInt("server.tls_min_version"),
		TLSMaxVersion:   config.GetInt("server.tls_max_version"),
		RootCaCerts:     config.GetStringSlice("server.tls_ca"),
		RootCaCertsFile: config.GetString("server.tls_ca_file"),
		Cert:            config.GetStringSlice("server.tls_cert"),
		CertFile:        config.GetString("server.tls_cert_file"),
		Key:             config.GetStringSlice("server.tls_key"),
		KeyFile:         config.GetString("server.tls_key_file"),
	})
	if err != nil {
		log.Fatalf("cannot read TLS config: %v", err)
	}
	return tlsc
}

func (s *SPAServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now().UTC()
	status := http.StatusOK
	defer func() {
		log.Infof("%s", s.logAccess(w, r, start, status))
	}()

	// handle CSP log
	switch r.Method {
	case http.MethodPost:
		if len(s.cfg.CspLog) > 0 && r.URL.Path == s.cfg.CspLog {
			// log CSP body
			body, _ := ioutil.ReadAll(r.Body)
			log.Info(string(body))
			w.WriteHeader(status)
			return
		}
	case http.MethodGet:
		// regular file access
	default:
		status = http.StatusMethodNotAllowed
		http.Error(w, http.StatusText(status), status)
		return
	}

	// strip base path or return 404
	fullname := strings.TrimPrefix(r.URL.Path, s.cfg.Base)
	if len(s.cfg.Base) > 0 && len(fullname) == len(r.URL.Path) {
		status = http.StatusNotFound
		http.NotFound(w, r)
		return
	}

	// try opening file
	// - may return an error when file exists but is not readable
	// - may return an index file as fallback
	// - may return a cached file
	f, name, err := s.TryFile(r, fullname)
	if err != nil {
		switch true {
		case os.IsNotExist(err):
			status = http.StatusNotFound
			http.NotFound(w, r)
		case os.IsPermission(err):
			status = http.StatusForbidden
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		default:
			log.Errorf("Opening file %s: %v", fullname[:32], err)
			status = http.StatusInternalServerError
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}
	// close file when done (a cached file will rewind) and use a func to
	// capture f because we may overwrite it below
	defer func() {
		f.Close()
	}()
	fi, _ := f.Stat()

	if !IsCached(f) {
		if cf, err := NewCachedFile(f); err == nil {
			f.Close()
			// try replace template variables
			if s.cfg.Tpl.Enable && s.cfg.Tpl.Match != nil && s.cfg.Tpl.Match.MatchString(name) {
				log.Debugf("Replacing templates in file %s", name)
				cf.ReplaceTemplates()
			}
			f = cf
			fi, _ = f.Stat()
			log.Debugf("Caching file %s", name)
			s.cache[name] = f
		} else if err != io.ErrShortBuffer {
			switch true {
			case os.IsNotExist(err):
				status = http.StatusNotFound
				http.NotFound(w, r)
			case os.IsPermission(err):
				status = http.StatusForbidden
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			default:
				log.Errorf("Caching file %s: %v", fullname[:32], err)
				status = http.StatusInternalServerError
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		} else {
			log.Warnf("Caching file %s failed: %v", name, err)
		}
		// don't cache or template-replace files on error (they may be too big to cache)
	}

	// write response headers
	s.WriteHeaders(w, r, f, start)

	// send file
	http.ServeContent(w, r, name, fi.ModTime(), f)
}

func (s *SPAServer) WriteHeaders(w http.ResponseWriter, r *http.Request, f http.File, start time.Time) {
	fi, _ := f.Stat()
	name := fi.Name()
	h := w.Header()

	// set cache headers based on filename and rules
	if s.cfg.Cache.Enable {
		rule := CacheRule{
			Expires: s.cfg.Cache.Expires,
			Control: s.cfg.Cache.Control,
		}
		for _, v := range s.cfg.Cache.Rules {
			if len(v.Filename) > 0 && v.Filename == name {
				log.Debugf("Using filename cache rule %#v", v)
				rule = v
				break
			}
			if v.Regexp != nil && v.Regexp.MatchString(name) {
				log.Debugf("Using regexp cache rule %#v", v)
				rule = v
				break
			}
		}
		if !rule.Ignore {
			if rule.NoCache {
				w.Header().Set("Cache-Control", "max-age=0, no-cache, no-store, must-revalidate")
				w.Header().Set("Pragma", "no-cache")
				w.Header().Set("Expires", start.Format(http.TimeFormat))
			} else {
				w.Header().Set("Cache-Control", rule.Control)
				w.Header().Set("Expires", start.Add(rule.Expires).Format(http.TimeFormat))
			}
		}
	}

	// set extra headers
	rid := r.Header.Get("X-Request-Id")
	if rid == "" {
		rid = "SV-" + <-idStream
	}
	h.Set("X-Request-Id", rid)

	for n, v := range s.headers {
		h.Add(n, v)
	}
}

func (s *SPAServer) TryFile(r *http.Request, name string) (http.File, string, error) {
	// lookup cache
	log.Debugf("Try cache lookup for file %s", s.cfg.Root+name)
	f, ok := s.cache[name]
	if ok {
		return f, name, nil
	}

	// check if file exists
	log.Debugf("Try opening file %s", s.cfg.Root+name)
	f, err := s.root.Open(name)
	if err == nil {
		fi, _ := f.Stat()
		if !fi.IsDir() {
			return f, name, nil
		}
		f.Close()
	} else if !os.IsNotExist(err) {
		return nil, name, err
	}

	// try with .html extension if missing
	if !strings.HasSuffix(name, ".html") {
		extname := name + ".html"
		log.Debugf("Try opening file %s", s.cfg.Root+extname)
		f, err := s.root.Open(extname)
		if err == nil {
			fi, _ := f.Stat()
			if !fi.IsDir() {
				return f, extname, nil
			}
			f.Close()
		} else if !os.IsNotExist(err) {
			return nil, extname, err
		}
	}

	// try lang-specific *-index.html matches first
	// en-US,en;q=0.5
	langs := strings.Split(r.Header.Get("Accept-Language"), ";")[0]
	if len(langs) > 0 {
		for _, v := range strings.Split(langs, ",") {
			v = strings.ToLower(strings.TrimSpace(v))
			name := "/" + v + "-" + s.cfg.Index
			log.Debugf("Try cache lookup for file %s", s.cfg.Root+name)
			f, ok := s.cache[name]
			if ok {
				return f, name, nil
			}
			log.Debugf("Try opening file %s", s.cfg.Root+name)
			f, err = s.root.Open(name)
			if err == nil {
				fi, _ := f.Stat()
				if !fi.IsDir() {
					return f, name, nil
				}
				f.Close()
			} else if !os.IsNotExist(err) {
				return nil, name, err
			}
		}
	}

	// fallback to index.html
	name = "/" + s.cfg.Index
	log.Debugf("Try cache lookup for file %s", s.cfg.Root+name)
	f, ok = s.cache[name]
	if ok {
		return f, name, nil
	}
	log.Debugf("Try opening file %s", s.cfg.Root+name)
	f, err = s.root.Open(name)
	return f, name, err
}

type AccessLog struct {
	Time          time.Time `json:"time"`
	RemoteAddr    string    `json:"remote_addr"`
	Host          string    `json:"host"`
	Request       string    `json:"request"`
	RequestMethod string    `json:"request_method"`
	Referrer      string    `json:"referrer"`
	UserAgent     string    `json:"user_agent"`
	SslProtocol   string    `json:"ssl_protocol"`
	SslCipher     string    `json:"ssl_cipher"`
	Status        int       `json:"status"`
	BodyBytesSent int       `json:"body_bytes_sent"`
	ContentType   string    `json:"content_type"`
	RequestId     string    `json:"request_id"`
	RequestTime   float64   `json:"request_time"`
}

func (s *SPAServer) logAccess(w http.ResponseWriter, r *http.Request, start time.Time, status int) string {
	// get real IP behind Docker Interface X-Real-IP or X-Forwarded-For
	remote := r.Header.Get("X-Real-Ip")
	if remote == "" {
		remote = r.Header.Get("X-Forwarded-For")
	}
	if remote == "" {
		remote, _, _ = net.SplitHostPort(r.RemoteAddr)
	}

	l := AccessLog{
		Time:          start,
		RemoteAddr:    remote,
		Host:          r.Host,
		Request:       strings.Join([]string{r.Method, r.URL.Path, r.Proto}, " "),
		RequestMethod: r.Method,
		Referrer:      r.Header.Get("Referer"),
		UserAgent:     r.Header.Get("User-Agent"),
	}

	if r.TLS != nil {
		l.SslProtocol = TLSVersionString(r.TLS.Version)
	} else {
		l.SslProtocol = "-"
	}

	// TODO: wait for TLS cipher names to be exported in Go 1.13 or 1.14
	l.SslCipher = "-"
	l.Status = status
	l.BodyBytesSent, _ = strconv.Atoi(w.Header().Get("Content-Length"))
	l.ContentType = w.Header().Get("Content-Type")
	l.RequestId = w.Header().Get("X-Request-ID")
	l.RequestTime = float64(time.Since(start).Truncate(time.Microsecond)) / float64(time.Second)

	buf, _ := json.Marshal(l)
	return string(buf)
}
