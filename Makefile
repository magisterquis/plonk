# Makefile
# Build plonk
# By J. Stuart McMurray
# Created 20230429
# Last Modified 20250504

BINNAME       != basename $$(pwd)
GOBUILDFLAGS   = -trimpath -ldflags "-w -s"
GOTESTFLAGS   += -timeout 3s
SHMORESUBR     = t/shmore.subr
SHMOREURL      = https://raw.githubusercontent.com/magisterquis/shmore/refs/heads/master/shmore.subr
TESTMANYCOUNT  =100
VETFLAGS       =-printf.funcs 'debugf,errorf,errorlogf,logf,printf'

.PHONY: all build test gotest provetest help install update clean

all: test build ## Build ALL the things (default)

${BINNAME}:
	go build ${GOBUILDFLAGS} -o ${BINNAME}

build: ${BINNAME}

test: gotest provetest ## Run ALL the tests

gotest: ## Run go-specific tests
	go test ${GOBUILDFLAGS} ${GOTESTFLAGS} ./...
	go vet ${GOBUILDFLAGS} ${VETFLAGS} ./...
	staticcheck ./...
	[ -z "$$(go fix -diff)" ]
	go run ${GOBUILDFLAGS} . -h 2>&1 |\
	awk '\
		/^Options:$$|MQD DEBUG PACKAGE LOADED$$/\
			{ exit }\
		/^Usage: /\
			{ sub(/^Usage: [^[:space:]]+\//, "Usage: ") }\
		/.{80,}/\
			{ print "Long usage line: " $$0; exit 1 }\
	'

longtest: ## Run Go tests 100 times
	go test -timeout 10s -count ${TESTMANYCOUNT} -short -failfast ${BUILDFLAGS} ./...

provetest: ## Run tests with prove(1) if ./t exists
.if exists(./t/)
	prove -It --directives
.endif

update: ## Fetch the latest Shmore and up-to-date Go things
	curl\
		--fail\
		--show-error\
		--silent\
		--output ${SHMORESUBR}.new\
		${SHMOREURL}
	diff -q ${SHMORESUBR} ${SHMORESUBR}.new >/dev/null &&\
		rm ${SHMORESUBR}.new ||\
		mv ${SHMORESUBR}.new ${SHMORESUBR}
	go get -t -u go ./...
	go mod tidy

install: ## Install to GOBIN ($GOPATH/bin or $HOME/go/bin)
	go install ${GOBUILDFLAGS}

clean: ## Remove built things
	rm -rf ${BINNAME}

help: .NOTMAIN ## This help
	@perl -ne '/^(\S+?):+.*?##\s*(.*)/&&print"$$1\t-\t$$2\n"' \
		${MAKEFILE_LIST} | column -ts "$$(printf "\t")"
