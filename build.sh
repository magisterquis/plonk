#!/bin/ksh
#
# build.sh
# Build plonk for a few platforms
# By J. Stuart McMurray
# Created 20230228
# Last Modified 20230523

set -e

if [[ "clean" == "$1" ]]; then
        for f in plonk plonk-*-*; do
                if [[ -f "$f" ]]; then
                        set -x
                        rm "$f"
                        set +x
                fi
        done
        exit 0
fi

(
        set -x
        go version
        go test
        go vet
        staticcheck
        go generate
        go build -trimpath -ldflags "-w -s"
)

build() {
        export GOOS
        N="$(basename $(pwd))-$(go env GOOS)-$(go env GOARCH)"
        set -x
        go build -trimpath -ldflags "-w -s" -o "$N"
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
