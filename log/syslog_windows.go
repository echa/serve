// Copyright (c) 2018-2019 KIDTSUNAMI
// Author: alex@kidtsunami.com

package log

import (
	"log"
	"os"
)

// no syslog on windows, write to stdout
func NewSyslog(config *Config) *Backend {
	return &Backend{config.Level, log.New(os.Stdout, "", config.Flags), ""}
}
