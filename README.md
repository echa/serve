SPAng - Dockerize Single-Page Javascript Apps
=============================================

A fast Go fileserver with a focus on **versatility** and **security**. Can be used stand-alone or behind a reverse proxy / load balancer. The main intention is to serve SPA assets from within Docker containers.

`spang` was created out of the need to have a reliable light-weight shipping method for SPA apps in Docker. The alternative - mounting assets from data-only Docker containers into a webserver container - is hard to manage because it relies on a special Docker feature that copies files from an exported volume to the host. Updating such a data-only container requires to remove it manually, remove the volume manually and restarting the webserver to bind-mount the new volume from the new container.

Works similar to Nginx `try_files` directive with some extra features and configuration options.

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

Configuration works via a JSON config file (default location is `/etc/spang/config.json` or `config.json` in CWD) and environment variables prefixed `SPANG_` (ENV takes precedence over config file settings).

Examples below contain defaults values.

```json
{
	"server": {
		// interface to bind to, env SPANG_SERVER_ADDR
		"addr": "0.0.0.0",
		// port number to bind to, env SPANG_SERVER_PORT
		"port": 8000,
		// protocol schema, either http or https, env SPANG_SERVER_SCHEME
		"scheme": "http",
		// filesystem root directory, env SPANG_SERVER_ROOT
		"root": "/var/www",
		// base URL (optional), env SPANG_SERVER_BASE
		"base": "",
		// app index file name, env SPANG_SERVER_INDEX
		"index": "index.html"
	}
}
```

### TLS Configuration

TLS is optional and will be enabled when you choose `https` as server scheme.

```json
{
	"server": {
		// server name for SNI (optional), env SPANG_SERVER_NAME
		"name": "",
		// TLS minimum version (e.g. 0 for TLSv1.0 to 3 for TLSv1.3), SPANG_SERVER_TLS_MIN_VERSION
		"tls_min_version": 3,
		// TLS maximum version (e.g. 0 for TLSv1.0 to 3 for TLSv1.3), SPANG_SERVER_TLS_MAX_VERSION
		"tls_max_version": 3,
		// TLS CA as PEM (multi-line strings will be concatenated), SPANG_SERVER_TLS_CA
		"tls_ca": [],
		// TLS CA file in PEM format, SPANG_SERVER_TLS_CA_FILE
		"tls_ca_file": "",
		// TLS Server Cert in PEM format (multi-line strings will be concatenated), SPANG_SERVER_TLS_CERT
		"tls_cert": [],
		// TLS Server Cert file in PEM format, SPANG_SERVER_TLS_CERT_FILE
		"tls_cert_file": "",
		// TLS Server Key in PEM format (multi-line strings will be concatenated), SPANG_SERVER_TLS_KEY
		"tls_key": [],
		// TLS Server Key file in PEM format, SPANG_SERVER_TLS_KEY_FILE
		"tls_key_file": ""
	}
}
```

### Config and Secrets Injection

When serving files, `spang` can scan for placeholders and replace them with the contents of environment variables on the fly. This way you can inject URLs, API keys, tokens, identifiers, and any kind of configuration settings into your Javascript and HTML files. That allows you to deploy the same docker images across integration testing, staging and production.

This feature is enabled by default. Don't use the same characters for start and end delimiters.

```json
	"template": {
		// enables or disables injection, env SPANG_TEMPLATE_ENABLE
		"enable": true,
		// template start delimiter, env SPANG_TEMPLATE_LEFT
		"left": "[[",
		// template end delimiter, env SPANG_TEMPLATE_RIGHT
		"right": "]]",
		// Go regexp to select files for injection, env SPANG_TEMPLATE_MATCH
		"match": "\\.(html|js)$",
		// max file size (helps prevent memory pressure), ebv SPANG_TEMPLATE_MAXSIZE
		"maxsize": 16777216
	}
```


### Multi-Language Index Support

`spang` can serve different `index.html` files based on the contents of the `Accept-Language` request header by trying different filenames with language used as prefix. Remember that you can change the name of the served index file in the server section or via `SPANG_SERVER_INDEX`.

Example for filename composition for default `index.html`:

```
Accept-Language: en-US,en;q=0.5

# first tried filename
en-us-index.html

# second tried filename
en-index.html

# generic fallback
index.html
```

### Controlling HTTP Caching

To control how `spang` returns HTTP cache headers you can specify multiple cache rules. This feature is enabled by default and will allow public caching of all files for 30 seconds.

```json
	"cache": {
		// enables or disabled cache headers, env SPANG_CACHE_ENABLE
		"enable": true,
		// set default cache lifetime, env SPANG_CACHE_EXIRES
		"expires": "30s",
		// set default cache policy, env SPANG_CACHE_CONTROL
		"control": "public",
		// define multiple rules to overwrite the default policy (config file only, NO env!)
		"rules": [{
			// specify a regexp to match files, i.e. for all index.html files
			"regexp": "\\.*index.html$",
			// send cache-control `max-age=0, no-cache, no-store, must-revalidate`
			"nocache": true,
			// do not send cache headers at all
			"ignore": true
		},{
			// specify a regexp to match files, i.e. all asset types
			"regexp": "\\.(js|css|png|jpg|jpeg|svg|ico|woff|ttf|eot|otf)$",
			// set a very long expiry time (e.g. 10 years)
			"expires": "87600h",
			// set an infinite expiry policy
			"control": "public, max-age=31536000, immutable",
		}]
	}
```


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

