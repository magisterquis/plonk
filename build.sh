#!/bin/ksh
#
# build.sh
# Build plonk for a few platforms
# By J. Stuart McMurray
# Created 20230228
# Last Modified 20230228

set -e

set -x
go version
go test
go vet
go generate
go build -trimpath
set +x

build() {
        export GOOS
        N="$(basename $(pwd))-$(go env GOOS)-$(go env GOARCH)"
        set -x
        go build -trimpath -o "$N"
        set +x
}

# Build for newer Macs
export GOARCH=arm64
for GOOS in darwin; do
        build
done

# Common platforms
export GOARCH=amd64
for GOOS in darwin linux openbsd; do
        build
done
