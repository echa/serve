// Copyright (c) 2019 KIDTSUNAMI
// Author: alex@kidtsunami.com

// Single-page Javascript App Server
// as drop-in replacement for nginx `try_files $uri $uri/ /index.html;`
//
// Features
//
// - HTTP/1.1 and HTTP/2.0 support
// - TLS server support
// - multi-language index.html from Accept-Language header
// - template replacement from env variables for safe secrets injection
// - custom HTTP headers
// - custom HTTP cache settings
// - configurable access logs
// - CSP endpoint and log

package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/echa/spang/config"
	"github.com/echa/spang/log"
	"github.com/echa/spang/server"
)

var (
	flags = flag.NewFlagSet("spang", flag.ContinueOnError)
	// verbosity levels
	vquiet  bool
	verbose bool
	vdebug  bool
	vtrace  bool
	vstats  int
)

func init() {
	flags.Usage = func() {}
	flags.BoolVar(&verbose, "v", false, "be verbose")
	flags.BoolVar(&vquiet, "q", false, "be quiet")
	flags.BoolVar(&vdebug, "d", false, "debug mode")
	flags.BoolVar(&vtrace, "t", false, "trace mode")

	// defaults
	config.SetEnvPrefix(config.APP_PREFIX)
	config.SetDefault("logging.flags", "date,time")
	config.SetDefault("logging.backend", "stdout")
	config.SetDefault("logging.level", "warn")
}

func main() {
	// parse cmdline flags
	if err := flags.Parse(os.Args[1:]); err != nil {
		if err == flag.ErrHelp {
			fmt.Println("SPAng Server")
			flags.PrintDefaults()
			os.Exit(0)
		}
		log.Fatal(err)
	}
	// read config file
	realconf := config.ConfigName()
	log.Info("Using config file ", realconf)
	if _, err := os.Stat(realconf); err == nil {
		if err := config.ReadConfigFile(); err != nil {
			fmt.Printf("Could not read %s: %v\n", realconf, err)
			os.Exit(1)
		}
	} else {
		log.Warn("Missing config file, using default values.")
	}

	// change log level
	switch true {
	case vquiet:
		config.Set("logging.level", "error")
	case vtrace:
		config.Set("logging.level", "trace")
	case vdebug:
		config.Set("logging.level", "debug")
	case verbose:
		config.Set("logging.level", "info")
	}
	// setup logging
	cfg := log.NewConfig()
	cfg.Level = log.ParseLevel(config.GetString("logging.level"))
	cfg.Flags = log.ParseFlags(config.GetString("logging.flags"))
	cfg.Backend = config.GetString("logging.backend")
	cfg.Filename = config.GetString("logging.filename")
	cfg.Addr = config.GetString("logging.syslog.address")
	cfg.Facility = config.GetString("logging.syslog.facility")
	cfg.Ident = config.GetString("logging.syslog.ident")
	cfg.FileMode = os.FileMode(config.GetInt("logging.filemode"))
	log.Init(cfg)

	// run
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errch := make(chan error, 1)
	var wg sync.WaitGroup
	go func() {
		wg.Add(1)
		defer wg.Done()
		errch <- serve(ctx)
	}()

	// wait for Ctrl-C
	stop := make(chan os.Signal, 1)
	signal.Notify(stop,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)

	select {
	case err := <-errch:
		if err != nil {
			log.Error(err)
		}
		return nil
	case <-stop:
		cancel()
		wg.Wait()
		if err := <-errch; err != nil {
			log.Error(err)
		} else {
			log.Info("Done.")
		}
	}
	return nil
}

func serve(ctx context.Context) error {
	spa, err := server.NewSPAServer()
	if err != nil {
		return err
	}

	s := &http.Server{
		Addr:              spa.Address(),
		TLSConfig:         spa.TLS(),
		Handler:           spa,
		ReadHeaderTimeout: config.GetDuration("server.header_timeout"),
		ReadTimeout:       config.GetDuration("server.read_timeout"),
		WriteTimeout:      config.GetDuration("server.write_timeout"),
		IdleTimeout:       config.GetDuration("server.idle_timeout"),
		MaxHeaderBytes:    1 << 20,
		ErrorLog:          log.Log.Logger(),
	}

	log.Infof("Starting HTTP/SPAng server at %s", s.Addr)
	log.Infof("Serving SPA from directory %s", config.GetString("server.root"))
	errch := make(chan error)
	go func() {
		errch <- s.ListenAndServe()
	}()
	select {
	case err := <-errch:
		return err
	case <-ctx.Done():
		log.Info("Stopping HTTP/SPAng server.")
		ctx2, cancel := context.WithCancel(context.Background())
		if tm := config.GetDuration("server.shutdown_timeout"); tm > 0 {
			ctx2, cancel = context.WithTimeout(ctx2, tm)
			defer cancel()
		}
		if err := s.Shutdown(ctx2); err != nil {
			return err
		}
	}
	return nil
}
