PROJECT := go-driver
SCRIPTDIR := $(shell pwd)
ROOTDIR := $(shell cd $(SCRIPTDIR) && pwd)

GOBUILDDIR := $(SCRIPTDIR)/.gobuild
GOVERSION := 1.7.5-alpine

ORGPATH := github.com/arangodb
ORGDIR := $(GOBUILDDIR)/src/$(ORGPATH)
REPONAME := $(PROJECT)
REPODIR := $(ORGDIR)/$(REPONAME)
REPOPATH := $(ORGPATH)/$(REPONAME)

SOURCES := $(shell find . -name '*.go')

.PHONY: all build clean run-tests

all: build

build: $(GOBUILDDIR) $(SOURCES)
	GOPATH=$(GOBUILDDIR) go build -v github.com/arangodb/go-driver github.com/arangodb/go-driver/http

clean:
	rm -Rf $(GOBUILDDIR)

$(GOBUILDDIR):
	@mkdir -p $(ORGDIR)
	@rm -f $(REPODIR) && ln -s ../../../.. $(REPODIR)

DBCONTAINER := $(PROJECT)-test-db

run-tests: build run-tests-single-no-auth 

run-tests-single-no-auth:
	@echo "Single server, no authentication"
	@-docker rm -f -v $(DBCONTAINER) &> /dev/null
	@docker run -d --name $(DBCONTAINER) \
		-e ARANGO_NO_AUTH=1 \
		arangodb:3.1.11
	@docker run \
		--rm \
		--net=container:$(DBCONTAINER) \
		-v $(ROOTDIR):/usr/code \
		-e GOPATH=/usr/code/.gobuild \
		-e TEST_ENDPOINTS=http://localhost:8529 \
		-w /usr/code/ \
		golang:$(GOVERSION) \
		go test -v $(REPOPATH) $(REPOPATH)/test
	@-docker rm -f -v $(DBCONTAINER) &> /dev/null
