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

all: build

build: $(GOBUILDDIR)
    GOPATH=$(GOBUILDDIR) go build github.com/arangodb/go-driver

$(GOBUILDDIR):
	@mkdir -p $(ORGDIR)
	@rm -f $(REPODIR) && ln -s ../../../.. $(REPODIR)

DBCONTAINER := $(PROJECT)-test-db

run-tests: build run-tests-single-no-auth

run-tests-single-no-auth:
	@echo "Single server, no authentication"
	@-docker rm -f -v $(DBCONTAINER) &> /dev/null
	@docker run -d --name $(DBCONTAINER) -p 8629:8529 \
		-e ARANGO_NO_AUTH=1 \
		arangodb:3.1
	@docker run \
		--rm \
		--link $(DBCONTAINER):db \
		-v $(ROOTDIR):/usr/code \
		-e GOPATH=/usr/code/.gobuild \
		-e TEST_ENDPOINTS=http://db:8529 \
		-w /usr/code/ \
		golang:$(GOVERSION) \
		go test -v $(REPOPATH)/test
	@-docker rm -f -v $(DBCONTAINER) &> /dev/null

