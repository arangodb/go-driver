PROJECT := go-driver
SCRIPTDIR := $(shell pwd)
ROOTDIR := $(shell cd $(SCRIPTDIR) && pwd)

GOBUILDDIR := $(SCRIPTDIR)/.gobuild
GOVERSION := 1.8-alpine

ifndef ARANGODB
	ARANGODB := arangodb/arangodb:3.1.16
	#ARANGODB := neunhoef/arangodb:3.2.devel-1
endif

TESTOPTIONS := 
ifdef VERBOSE
	TESTOPTIONS := -v
endif

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
TESTCONTAINER := $(PROJECT)-test

run-tests: run-tests-http run-tests-single run-tests-cluster

# Tests of HTTP package 
run-tests-http: $(GOBUILDDIR)
	@docker run \
		--rm \
		-v $(ROOTDIR):/usr/code \
		-e GOPATH=/usr/code/.gobuild \
		-w /usr/code/ \
		golang:$(GOVERSION) \
		go test $(TESTOPTIONS) $(REPOPATH)/http

# Single server tests 
run-tests-single: run-tests-single-with-auth run-tests-single-no-auth

run-tests-single-no-auth: $(GOBUILDDIR)
	@echo "Single server, no authentication"
	@-docker rm -f -v $(DBCONTAINER) $(TESTCONTAINER) &> /dev/null
	@docker run -d --name $(DBCONTAINER) \
		-e ARANGO_NO_AUTH=1 \
		$(ARANGODB)
	@docker run \
		--name=$(TESTCONTAINER) \
		--net=container:$(DBCONTAINER) \
		-v $(ROOTDIR):/usr/code \
		-e GOPATH=/usr/code/.gobuild \
		-e TEST_ENDPOINTS=http://localhost:8529 \
		-w /usr/code/ \
		golang:$(GOVERSION) \
		go test $(TESTOPTIONS) $(REPOPATH) $(REPOPATH)/test
	@docker rm -f -v $(TESTCONTAINER) &> /dev/null
	@docker rm -f -v $(DBCONTAINER) &> /dev/null
	@sleep 3

run-tests-single-with-auth: $(GOBUILDDIR)
	@echo "Single server, with authentication"
	@-docker rm -f -v $(DBCONTAINER) $(TESTCONTAINER) &> /dev/null
	@docker run -d --name $(DBCONTAINER) \
		-e ARANGO_ROOT_PASSWORD=rootpw \
		$(ARANGODB)
	@docker run \
		--name=$(TESTCONTAINER) \
		--net=container:$(DBCONTAINER) \
		-v $(ROOTDIR):/usr/code \
		-e GOPATH=/usr/code/.gobuild \
		-e TEST_ENDPOINTS=http://localhost:8529 \
		-e TEST_AUTHENTICATION=basic:root:rootpw \
		-w /usr/code/ \
		golang:$(GOVERSION) \
		go test -tags auth $(TESTOPTIONS) $(REPOPATH)/test
	@docker rm -f -v $(TESTCONTAINER) &> /dev/null
	@docker rm -f -v $(DBCONTAINER) &> /dev/null
	@sleep 3

# Cluster mode tests
run-tests-cluster: run-tests-cluster-no-auth run-tests-cluster-with-auth run-tests-cluster-ssl

run-tests-cluster-no-auth: $(GOBUILDDIR)
	@echo "Cluster server, no authentication"
	@PROJECT=$(PROJECT) ARANGODB=$(ARANGODB) $(ROOTDIR)/test/cluster.sh start
	docker run \
		--rm \
		--net=host \
		-v $(ROOTDIR):/usr/code \
		-e GOPATH=/usr/code/.gobuild \
		-e TEST_ENDPOINTS=http://localhost:7002 \
		-w /usr/code/ \
		golang:$(GOVERSION) \
		go test $(TESTOPTIONS) $(REPOPATH)/test
	@PROJECT=$(PROJECT) ARANGODB=$(ARANGODB) $(ROOTDIR)/test/cluster.sh cleanup

run-tests-cluster-with-auth: $(GOBUILDDIR)
	@echo "Cluster server, with authentication"
	@PROJECT=$(PROJECT) ARANGODB=$(ARANGODB) TMPDIR=${GOBUILDDIR} JWTSECRET=testing $(ROOTDIR)/test/cluster.sh start
	docker run \
		--rm \
		--net=host \
		-v $(ROOTDIR):/usr/code \
		-e GOPATH=/usr/code/.gobuild \
		-e TEST_ENDPOINTS=http://localhost:7002 \
		-e TEST_AUTHENTICATION=basic:root: \
		-w /usr/code/ \
		golang:$(GOVERSION) \
		go test -tags auth $(TESTOPTIONS) $(REPOPATH)/test
	@PROJECT=$(PROJECT) ARANGODB=$(ARANGODB) $(ROOTDIR)/test/cluster.sh cleanup

run-tests-cluster-ssl: $(GOBUILDDIR)
	@echo "Cluster server, SSL, with authentication"
	@PROJECT=$(PROJECT) ARANGODB=$(ARANGODB) SSL=auto TMPDIR=${GOBUILDDIR} JWTSECRET=testing $(ROOTDIR)/test/cluster.sh start
	docker run \
		--rm \
		--net=host \
		-v $(ROOTDIR):/usr/code \
		-e GOPATH=/usr/code/.gobuild \
		-e TEST_ENDPOINTS=https://localhost:7002 \
		-e TEST_AUTHENTICATION=basic:root: \
		-w /usr/code/ \
		golang:$(GOVERSION) \
		go test -tags auth $(TESTOPTIONS) $(REPOPATH)/test
	@PROJECT=$(PROJECT) ARANGODB=$(ARANGODB) $(ROOTDIR)/test/cluster.sh cleanup

run-tests-cluster-failover: $(GOBUILDDIR)
	# Note that we use 127.0.0.1:7002.. as endpoints, so we force using IPv4
	# This is essential since we only block IPv4 ports in the test.
	@echo "Cluster server, failover, no authentication"
	@PROJECT=$(PROJECT) ARANGODB=$(ARANGODB) $(ROOTDIR)/test/cluster.sh start
	GOPATH=$(GOBUILDDIR) go get github.com/coreos/go-iptables/iptables
	docker run \
		--rm \
		--net=host \
		--privileged \
		-v $(ROOTDIR):/usr/code \
		-e GOPATH=/usr/code/.gobuild \
		-e TEST_ENDPOINTS=http://127.0.0.1:7002,http://127.0.0.1:7007,http://127.0.0.1:7012 \
		-e TEST_AUTHENTICATION=basic:root: \
		-w /usr/code/ \
		golang:$(GOVERSION) \
		/bin/sh -c 'apk add -U iptables && go test -run ".*Failover.*" -tags failover $(TESTOPTIONS) $(REPOPATH)/test'
	@PROJECT=$(PROJECT) ARANGODB=$(ARANGODB) $(ROOTDIR)/test/cluster.sh cleanup

run-tests-cluster-cleanup:
	@PROJECT=$(PROJECT) ARANGODB=$(ARANGODB) $(ROOTDIR)/test/cluster.sh cleanup

# Benchmarks
run-benchmarks-single-no-auth: $(GOBUILDDIR)
	@echo "Single server, no authentication"
	@-docker rm -f -v $(DBCONTAINER) $(TESTCONTAINER) &> /dev/null
	@docker run -d --name $(DBCONTAINER) \
		-e ARANGO_NO_AUTH=1 \
		$(ARANGODB)
	@docker run \
		--name=$(TESTCONTAINER) \
		--net=container:$(DBCONTAINER) \
		-v $(ROOTDIR):/usr/code \
		-e GOPATH=/usr/code/.gobuild \
		-e TEST_ENDPOINTS=http://localhost:8529 \
		-w /usr/code/ \
		golang:$(GOVERSION) \
		go test $(TESTOPTIONS) -bench=. -run=notests -cpu=1,2,4 $(REPOPATH)/test
	@docker rm -f -v $(TESTCONTAINER) &> /dev/null
	@docker rm -f -v $(DBCONTAINER) &> /dev/null
	@sleep 3
