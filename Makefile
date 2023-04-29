# Makefile
# Build Plonk
# By J. Stuart McMurray
# Created 20230429
# Last Modified 20230429

BINNAME=plonk

all: test build

test:
	go test
	go vet
	staticcheck
	
build:
	go build -trimpath -ldflags="-w -s" -o ${BINNAME}

clean:
	rm -f ${BINNAME}
