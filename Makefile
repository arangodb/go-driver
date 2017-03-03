PROJECT := go-driver
SCRIPTDIR := $(shell pwd)
ROOTDIR := $(shell cd $(SCRIPTDIR) && pwd)

GOBUILDDIR := $(SCRIPTDIR)/.gobuild
GOVERSION := 1.8-alpine

ARANGODB := arangodb:3.1.12
#ARANGODB := neunhoef/arangodb:3.2.devel-1

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

run-tests: run-tests-single-with-auth run-tests-single-no-auth

run-tests-single-no-auth:
	@echo "Single server, no authentication"
	@-docker rm -f -v $(DBCONTAINER) &> /dev/null
	@docker run -d --name $(DBCONTAINER) \
		-e ARANGO_NO_AUTH=1 \
		$(ARANGODB)
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

run-tests-single-with-auth:
	@echo "Single server, with authentication"
	@-docker rm -f -v $(DBCONTAINER) &> /dev/null
	@docker run -d --name $(DBCONTAINER) \
		-e ARANGO_ROOT_PASSWORD=rootpw \
		$(ARANGODB)
	@docker run \
		--rm \
		--net=container:$(DBCONTAINER) \
		-v $(ROOTDIR):/usr/code \
		-e GOPATH=/usr/code/.gobuild \
		-e TEST_ENDPOINTS=http://localhost:8529 \
		-e TEST_AUTHENTICATION=basic:root:rootpw \
		-w /usr/code/ \
		golang:$(GOVERSION) \
		go test -tags auth -v $(REPOPATH)/test
	@-docker rm -f -v $(DBCONTAINER) &> /dev/null
