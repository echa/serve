# go-spa build container
#
FROM            golang:alpine as builder
ARG             BUILD_ID
LABEL           stage=serve-builder
LABEL           build=$BUILD_ID

WORKDIR         /go/src/github.com/echa/serve
COPY            . .
RUN             apk --no-cache add git binutils
RUN             go mod download
RUN             CGO_ENABLED=0 go build -a -ldflags="-s -w" -o /serve .
RUN             strip /serve

FROM            alpine:latest
MAINTAINER      Alexander Eichhorn <alex@kidtsunami.com>

ARG             BUILD_TARGET
ARG             BUILD_VERSION
ARG             BUILD_DATE
ARG             BUILD_ID=unset

LABEL           SV_BUILD_VERSION=$BUILD_VERSION \
                SV_BUILD_ID=$BUILD_ID \
                SV_BUILD_DATE=$BUILD_DATE

ENV             SV_BUILD_VERSION=$BUILD_VERSION
ENV             SV_BUILD_ID=$BUILD_ID
ENV             SV_BUILD_DATE=$BUILD_DATE

ENV             SV_CONFIG_FILE /etc/serve/config.json
COPY            config.json /etc/serve/config.json
COPY            --from=builder /serve /usr/local/bin/serve
RUN             apk add --no-cache ca-certificates \
		  	    && addgroup www-data -g 500 \
			    && adduser -u 500 -D -h /var/www -S -s /sbin/nologin -G www-data www-data

WORKDIR         /var/www
USER            www-data
EXPOSE          8000
ENTRYPOINT      serve