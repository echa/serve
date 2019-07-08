SPAng - A fast Go fileserver for dockerizing Single-Page Javascript Apps
========================================================================

Go fileserver with a focus on **speed**, **versatility** and **security**. Can be used stand-alone or behind a reverse proxy / load balancer. The main intention is to use `spa` as light-weight version-controlled shipping method for SPA apps in Docker containers.

Works similar to Nginx `try_files $uri $uri/ /index.html;` directive with a bulkload of extra features and configuration options.

### Features

- HTTP/1.1 and HTTP/2.0 support
- TLS server support
- HTTP file server with auto mime-type detection
- serves multi-language index.html based on Accept-Language request header
- template replacement from ENV variables for safe secrets injection
- custom HTTP headers
- custom HTTP cache settings
- configurable access logs
- CSP report logging

### Main Server Configuration

Configuration works via a JSON config file (default location is `/etc/spa/config.json` or `config.json` in CWD) and environment variables prefixed `SPA_` (ENV takes precedence over config file settings).

Examples below contain defaults values.

```json
{
	"server": {
		// interface to bind to, env SPA_SERVER_ADDR
		"addr": "0.0.0.0",
		// port number to bind to, env SPA_SERVER_PORT
		"port": 8000,
		// protocol schema, either http or https, env SPA_SERVER_SCHEME
		"scheme": "http",
		// filesystem root directory, env SPA_SERVER_ROOT
		"root": "/var/www",
		// base URL (optional), env SPA_SERVER_BASE
		"base": "",
		// app index file name, env SPA_SERVER_INDEX
		"index": "index.html"
	}
}
```

### TLS Configuration

TLS is optional and will be enabled when you choose `https` as server scheme.

```json
{
	"server": {
		// server name for SNI (optional), env SPA_SERVER_NAME
		"name": "",
		// TLS minimum version (e.g. 0 for TLSv1.0 to 3 for TLSv1.3), SPA_SERVER_TLS_MIN_VERSION
		"tls_min_version": 3,
		// TLS maximum version (e.g. 0 for TLSv1.0 to 3 for TLSv1.3), SPA_SERVER_TLS_MAX_VERSION
		"tls_max_version": 3,
		// TLS CA as PEM (multi-line strings will be concatenated), SPA_SERVER_TLS_CA
		"tls_ca": [],
		// TLS CA file in PEM format, SPA_SERVER_TLS_CA_FILE
		"tls_ca_file": "",
		// TLS Server Cert in PEM format (multi-line strings will be concatenated), SPA_SERVER_TLS_CERT
		"tls_cert": [],
		// TLS Server Cert file in PEM format, SPA_SERVER_TLS_CERT_FILE
		"tls_cert_file": "",
		// TLS Server Key in PEM format (multi-line strings will be concatenated), SPA_SERVER_TLS_KEY
		"tls_key": [],
		// TLS Server Key file in PEM format, SPA_SERVER_TLS_KEY_FILE
		"tls_key_file": ""
	}
}
```

### Config and Secrets Injection

TODO

### Multi-Language Index Support

TODO

### Controlling HTTP Caching

TODO


### Setting Custom HTTP Headers

Additional response headers may be added under the `headers` key as key/values. They will be added to all served files.

```json
{
	"headers": {
		"key": "value"
	}
}
```

### How to build

You need Git and Go installed on your machine. No special dependencies required.

```
git clone https://github.com/echa/spang.git
cd spang && go build
```

### How to run

```
go run .
```

### License

The MIT License (MIT) Copyright (c) 2019 KIDTSUNAMI.

