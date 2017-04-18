PROJECT := go-driver
SCRIPTDIR := $(shell pwd)
ROOTDIR := $(shell cd $(SCRIPTDIR) && pwd)

GOBUILDDIR := $(SCRIPTDIR)/.gobuild
GOVERSION := 1.8-alpine

ifndef ARANGODB
	ARANGODB := arangodb/arangodb:3.1.17
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

# Test variables

DBCONTAINER := $(PROJECT)-test-db
TESTCONTAINER := $(PROJECT)-test

ifeq ("$(TEST_AUTH)", "none")
	ARANGOENV := -e ARANGO_NO_AUTH=1
	TEST_AUTHENTICATION := 
	TAGS := 
	TESTS := $(REPOPATH) $(REPOPATH)/test
else ifeq ("$(TEST_AUTH)", "rootpw")
	ARANGOENV := -e ARANGO_ROOT_PASSWORD=rootpw
	TEST_AUTHENTICATION := basic:root:rootpw
	TAGS := -tags auth
	TESTS := $(REPOPATH)/test
endif

ifeq ("$(TEST_MODE)", "single")
	TEST_NET := container:$(DBCONTAINER)
	TEST_ENDPOINTS := http://localhost:8529
else 
	TEST_NET := host
	TEST_ENDPOINTS := http://localhost:7002
ifeq ("$(TEST_AUTH)", "rootpw")
	CLUSTERENV := JWTSECRET=testing
	TEST_AUTHENTICATION := basic:root:
endif
ifeq ("$(TEST_SSL)", "auto")
	CLUSTERENV := SSL=auto $(CLUSTERENV)
	TEST_ENDPOINTS = https://localhost:7002
endif
endif

ifeq ("$(TEST_BENCHMARK)", "true")
	TAGS := -bench=. -run=notests -cpu=1,2,4
	TESTS := $(REPOPATH)/test
endif

.PHONY: all build clean run-tests

all: build

build: $(GOBUILDDIR) $(SOURCES)
	GOPATH=$(GOBUILDDIR) go build -v github.com/arangodb/go-driver github.com/arangodb/go-driver/http

clean:
	rm -Rf $(GOBUILDDIR)

$(GOBUILDDIR):
	@mkdir -p $(ORGDIR)
	@rm -f $(REPODIR) && ln -s ../../../.. $(REPODIR)
	GOPATH=$(GOBUILDDIR) go get github.com/arangodb/go-velocypack

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
run-tests-single: run-tests-single-json run-tests-single-vpack

run-tests-single-json: run-tests-single-json-with-auth run-tests-single-json-no-auth

run-tests-single-vpack: run-tests-single-vpack-with-auth run-tests-single-vpack-no-auth

run-tests-single-json-no-auth:
	@echo "Single server, HTTP+JSON, no authentication"
	@${MAKE} TEST_MODE="single" TEST_AUTH="none" TEST_CONTENT_TYPE="json" __run_tests

run-tests-single-vpack-no-auth:
	@echo "Single server, HTTP+Velocypack, no authentication"
	@${MAKE} TEST_MODE="single" TEST_AUTH="none" TEST_CONTENT_TYPE="vpack" __run_tests

run-tests-single-json-with-auth:
	@echo "Single server, HTTP+JSON, with authentication"
	@${MAKE} TEST_MODE="single" TEST_AUTH="rootpw" TEST_CONTENT_TYPE="json" __run_tests

run-tests-single-vpack-with-auth:
	@echo "Single server, HTTP+Velocypack, with authentication"
	@${MAKE} TEST_MODE="single" TEST_AUTH="rootpw" TEST_CONTENT_TYPE="vpack" __run_tests

# Cluster mode tests
run-tests-cluster: run-tests-cluster-json run-tests-cluster-vpack

run-tests-cluster-json: run-tests-cluster-json-no-auth run-tests-cluster-json-with-auth run-tests-cluster-json-ssl

run-tests-cluster-vpack: run-tests-cluster-vpack-no-auth run-tests-cluster-vpack-with-auth run-tests-cluster-vpack-ssl

run-tests-cluster-json-no-auth: $(GOBUILDDIR)
	@echo "Cluster server, JSON, no authentication"
	@${MAKE} TEST_MODE="cluster" TEST_AUTH="none" TEST_CONTENT_TYPE="json" __run_tests

run-tests-cluster-vpack-no-auth: $(GOBUILDDIR)
	@echo "Cluster server, Velocpack, no authentication"
	@${MAKE} TEST_MODE="cluster" TEST_AUTH="none" TEST_CONTENT_TYPE="vpack" __run_tests

run-tests-cluster-json-with-auth: $(GOBUILDDIR)
	@echo "Cluster server, with authentication"
	@${MAKE} TEST_MODE="cluster" TEST_AUTH="rootpw" TEST_CONTENT_TYPE="json" __run_tests

run-tests-cluster-vpack-with-auth: $(GOBUILDDIR)
	@echo "Cluster server, Velocypack, with authentication"
	@${MAKE} TEST_MODE="cluster" TEST_AUTH="rootpw" TEST_CONTENT_TYPE="vpack" __run_tests

run-tests-cluster-json-ssl: $(GOBUILDDIR)
	@echo "Cluster server, SSL, with authentication"
	@${MAKE} TEST_MODE="cluster" TEST_AUTH="rootpw" TEST_SSL="auto" TEST_CONTENT_TYPE="json" __run_tests

run-tests-cluster-vpack-ssl: $(GOBUILDDIR)
	@echo "Cluster server, Velocypack, SSL, with authentication"
	@${MAKE} TEST_MODE="cluster" TEST_AUTH="rootpw" TEST_SSL="auto" TEST_CONTENT_TYPE="vpack" __run_tests

# Internal test tasks
__run_tests: $(GOBUILDDIR) __test_prepare __test_go_test __test_cleanup

__test_go_test:
	docker run \
		--name=$(TESTCONTAINER) \
		--net=$(TEST_NET) \
		-v $(ROOTDIR):/usr/code \
		-e GOPATH=/usr/code/.gobuild \
		-e TEST_ENDPOINTS=$(TEST_ENDPOINTS) \
		-e TEST_AUTHENTICATION=$(TEST_AUTHENTICATION) \
		-e TEST_CONTENT_TYPE=$(TEST_CONTENT_TYPE) \
		-w /usr/code/ \
		golang:$(GOVERSION) \
		go test $(TAGS) $(TESTOPTIONS) $(TESTS)

__test_prepare:
ifeq ("$(TEST_MODE)", "single")
	@-docker rm -f -v $(DBCONTAINER) $(TESTCONTAINER) &> /dev/null
	docker run -d --name $(DBCONTAINER) \
		$(ARANGOENV) \
		$(ARANGODB)
else
	@-docker rm -f -v $(TESTCONTAINER) &> /dev/null
	@PROJECT=$(PROJECT) ARANGODB=$(ARANGODB) TMPDIR=${GOBUILDDIR} $(CLUSTERENV) $(ROOTDIR)/test/cluster.sh start
endif

__test_cleanup:
	@docker rm -f -v $(TESTCONTAINER) &> /dev/null
ifeq ("$(TEST_MODE)", "single")
	@docker rm -f -v $(DBCONTAINER) &> /dev/null
else
	@PROJECT=$(PROJECT) ARANGODB=$(ARANGODB) $(ROOTDIR)/test/cluster.sh cleanup
endif
	@sleep 3


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
run-benchmarks-single-json-no-auth: 
	@echo "Benchmarks: Single server, JSON no authentication"
	@${MAKE} TEST_MODE="single" TEST_AUTH="none" TEST_CONTENT_TYPE="json" TEST_BENCHMARK="true" __run_tests

run-benchmarks-single-vpack-no-auth: 
	@echo "Benchmarks: Single server, Velocypack, no authentication"
	@${MAKE} TEST_MODE="single" TEST_AUTH="none" TEST_CONTENT_TYPE="vpack" TEST_BENCHMARK="true" __run_tests
