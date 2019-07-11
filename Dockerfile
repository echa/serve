# go-spa build container
#
FROM            golang:1.12.6-alpine3.10 as builder
ARG             BUILD_ID
LABEL           stage=spang-builder
LABEL           build=$BUILD_ID

WORKDIR         /go/src/github.com/echa/spang
COPY            . .
RUN             CGO_ENABLED=0 go build -a -ldflags="-s -w" -o /spang .


FROM            alpine:3.10
MAINTAINER      Alexander Eichhorn <alex@kidtsunami.com>

ARG             BUILD_TARGET
ARG             BUILD_VERSION
ARG             BUILD_DATE
ARG             BUILD_ID=unset

LABEL           SPANG_BUILD_VERSION=$BUILD_VERSION \
                SPANG_BUILD_ID=$BUILD_ID \
                SPANG_BUILD_DATE=$BUILD_DATE

ENV             SPANG_BUILD_VERSION=$BUILD_VERSION
ENV             SPANG_BUILD_ID=$BUILD_ID
ENV             SPANG_BUILD_DATE=$BUILD_DATE

ENV             SPANG_CONFIG_FILE /etc/spang/config.json
COPY            config.json /etc/spang/config.json
COPY            --from=builder /spang /usr/local/bin/spang
RUN             apk add --no-cache ca-certificates \
		  	    && addgroup www-data -g 500 \
			    && adduser -u 500 -D -h /var/www -S -s /sbin/nologin -G www-data www-data

WORKDIR         /var/www
USER            www-data
EXPOSE          8000
ENTRYPOINT      spang