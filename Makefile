# Makefile
# Build Plonk
# By J. Stuart McMurray
# Created 20230429
# Last Modified 20230726

BINNAME=plonk

all: test build

test:
	go test ./...
	go vet ./...
	staticcheck ./...
	go run . -h 2>&1 |\
	awk '/.{80,}/ {print "Long usage line: " $$0; exit 1}\
		/^Options:/ {exit}'
	
build:
	go build -trimpath -ldflags="-w -s" -o ${BINNAME}

clean:
	rm -f ${BINNAME}
