// Copyright (c) 2018-2019 KIDTSUNAMI
// Author: alex@kidtsunami.com

package server

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/echa/spang/log"
)

var (
	ERootCAFailed = errors.New("failed to add Root CAs to certificate pool")
)

func TLSVersion(v int) uint16 {
	switch v {
	case 0:
		return tls.VersionTLS10
	case 1:
		return tls.VersionTLS11
	case 2:
		return tls.VersionTLS12
	case 3:
		return tls.VersionTLS13
	}
	return tls.VersionTLS13
}

func TLSVersionString(v uint16) string {
	switch v {
	case tls.VersionSSL30:
		return "SSLv3"
	case tls.VersionTLS10:
		return "TLSv1.0"
	case tls.VersionTLS11:
		return "TLSv1.1"
	case tls.VersionTLS12:
		return "TLSv1.2"
	case tls.VersionTLS13:
		return "TLSv1.3"
	default:
		return fmt.Sprintf("unknown TLS version %d", v)
	}
}

type TLSConfig struct {
	ServerName         string   `json:"server_name"`
	AllowInsecureCerts bool     `json:"disable_tls"`
	TLSMinVersion      int      `json:"tls_min_version"`
	TLSMaxVersion      int      `json:"tls_max_version"`
	RootCaCerts        []string `json:"tls_ca"`
	RootCaCertsFile    string   `json:"tls_ca_file"`
	Cert               []string `json:"tls_cert"`
	CertFile           string   `json:"tls_cert_file"`
	Key                []string `json:"tls_key"`
	KeyFile            string   `json:"tls_key_file"`
}

func NewTLSConfig(c TLSConfig) (*tls.Config, error) {
	if err := c.Check(); err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: c.AllowInsecureCerts,
		ServerName:         c.ServerName,
		MinVersion:         TLSVersion(c.TLSMinVersion),
		MaxVersion:         TLSVersion(c.TLSMaxVersion),
	}
	if len(c.RootCaCerts) > 0 {
		// load from config
		rootCAs := x509.NewCertPool()
		if !rootCAs.AppendCertsFromPEM([]byte(strings.Join(c.RootCaCerts, "\n"))) {
			return nil, ERootCAFailed
		}
		tlsConfig.RootCAs = rootCAs
	} else if len(c.RootCaCertsFile) > 0 {
		// load from file
		caCert, err := ioutil.ReadFile(c.RootCaCertsFile)
		if err != nil {
			return nil, fmt.Errorf("Could not load TLS CA [%s]: %v", c.RootCaCertsFile, err)
		}
		rootCAs := x509.NewCertPool()
		if !rootCAs.AppendCertsFromPEM(caCert) {
			return nil, ERootCAFailed
		}
		tlsConfig.RootCAs = rootCAs
	}
	if len(c.Cert) > 0 && len(c.Key) > 0 {
		// load from config
		cert, err := tls.X509KeyPair(
			[]byte(strings.Join(c.Cert, "\n")),
			[]byte(strings.Join(c.Key, "\n")),
		)
		if err != nil {
			return nil, err
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
		tlsConfig.BuildNameToCertificate()
	} else if len(c.CertFile) > 0 && len(c.KeyFile) > 0 {
		// load from file
		cert, err := tls.LoadX509KeyPair(c.CertFile, c.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("Could not load TLS client cert or key [%s]: %v", c.CertFile, err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
		tlsConfig.BuildNameToCertificate()
	}
	return tlsConfig, nil
}

func (cfg TLSConfig) Check() error {
	if len(cfg.RootCaCerts) == 0 && len(cfg.RootCaCertsFile) == 0 {
		log.Warn("empty Root CA cert chain")
	}

	if cfg.AllowInsecureCerts {
		log.Warn("accepting insecure certificates! This is dangerous!")
	}

	if cfg.TLSMinVersion < 3 {
		log.Warn("insecure TLS version! Use TLSv1.2 (3) or above.")
	}

	if len(cfg.Cert) > 0 && len(cfg.Key) == 0 {
		log.Warn("missing TLS key")
	}

	if len(cfg.Key) > 0 && len(cfg.Cert) == 0 {
		log.Warn("missing TLS cert")
	}

	if len(cfg.CertFile) > 0 && len(cfg.KeyFile) == 0 {
		log.Warn("missing TLS key file")
	}

	if len(cfg.KeyFile) > 0 && len(cfg.CertFile) == 0 {
		log.Warn("missing TLS cert file")
	}

	return nil
}
