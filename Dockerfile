# go-spa build container
#
FROM            golang:1.12.6-alpine3.10 as builder

WORKDIR         /go/src/github.com/echa/spang
COPY            . .
RUN             CGO_ENABLED=0 go build -a -ldflags="-s -w" -o /spang .


FROM            alpine:3.10
MAINTAINER      Alexander Eichhorn <alex@kidtsunami.com>

ENV             SPA_CONFIG_FILE /etc/spang/config.json
COPY            config.json /etc/spang/config.json
COPY            --from=builder /spang /usr/local/bin/spang
RUN             apk add --no-cache ca-certificates \
		  	    && addgroup www-data -g 500 \
			    && adduser -u 500 -D -h /var/www -S -s /sbin/nologin -G www-data www-data

WORKDIR         /var/www
USER            www-data
EXPOSE          8000
ENTRYPOINT      spang