#!/usr/bin/env bash
set -xe
go get -u -v golang.org/x/lint/golint
go get -u -v github.com/tools/godep
go get -u -v github.com/mitchellh/gox
go get -u -v github.com/jteeuwen/go-bindata/...
go get -u -v github.com/go-playground/overalls
go get -u -v github.com/tgulacsi/go/wrap
go get -u -v github.com/xizzat/go-elastictable
godep restore -v
