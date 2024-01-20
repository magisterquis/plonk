# Makefile
# Build Plonk
# By J. Stuart McMurray
# Created 20230429
# Last Modified 20240120

BINNAME!=basename $$(pwd)
BUILDFLAGS=-trimpath -ldflags "-w -s"
VETFLAGS=-printf.funcs 'debugf,errorf,erorrlogf,logf,printf'
TESTMANYCOUNT=100

all: test build

test:
	go test -timeout 3s ${BUILDFLAGS} ./...
	go vet  ${BUILDFLAGS} ${VETFLAGS} ./...
	staticcheck ./...
	go run ${BUILDFLAGS} . -h 2>&1 |\
	awk '\
		/^Options:$$|MQD DEBUG PACKAGE LOADED$$/ {exit}\
		/.{80,}/ {print "Long usage line: " $$0; exit 1}\
	'

longtest:
	go test -timeout 10s -count ${TESTMANYCOUNT} -short -failfast ${BUILDFLAGS} ./...
	
build:
	go build ${BUILDFLAGS} -o ${BINNAME}

install:
	go install ${BUILDFLAGS}

clean:
	rm -f ${BINNAME}
