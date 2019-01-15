PROJECT := go-driver
SCRIPTDIR := $(shell pwd)
ROOTDIR := $(shell cd $(SCRIPTDIR) && pwd)

GOBUILDDIR := $(SCRIPTDIR)/.gobuild
GOVERSION := 1.10.1-alpine
TMPDIR := $(GOBUILDDIR)

ifndef ARANGODB
	ARANGODB := arangodb/arangodb:latest
endif
ifndef STARTER
	STARTER := arangodb/arangodb-starter:latest
endif

ifndef TESTOPTIONS
	TESTOPTIONS := 
endif
ifdef VERBOSE
	TESTVERBOSEOPTIONS := -v
endif

ORGPATH := github.com/arangodb
ORGDIR := $(GOBUILDDIR)/src/$(ORGPATH)
REPONAME := $(PROJECT)
REPODIR := $(ORGDIR)/$(REPONAME)
REPOPATH := $(ORGPATH)/$(REPONAME)

SOURCES := $(shell find . -name '*.go')

# Test variables

ifndef TESTCONTAINER
	TESTCONTAINER := $(PROJECT)-test
endif
ifndef DBCONTAINER
	DBCONTAINER := $(TESTCONTAINER)-db
endif 

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
else ifeq ("$(TEST_AUTH)", "jwt")
	ARANGOENV := -e ARANGO_ROOT_PASSWORD=rootpw 
	TEST_AUTHENTICATION := jwt:root:rootpw
	TAGS := -tags auth
	TESTS := $(REPOPATH)/test
	JWTSECRET := testing
	JWTSECRETFILE := $(TMPDIR)/$(TESTCONTAINER)-jwtsecret
	ARANGOVOL := -v "$(JWTSECRETFILE):/jwtsecret"
	ARANGOARGS := --server.jwt-secret=/jwtsecret
endif

TEST_NET := container:$(TESTCONTAINER)-ns
TEST_ENDPOINTS := http://localhost:7001
TESTS := $(REPOPATH)/test
ifeq ("$(TEST_AUTH)", "rootpw")
	CLUSTERENV := JWTSECRET=testing
	TEST_AUTHENTICATION := basic:root:
endif
ifeq ("$(TEST_AUTH)", "jwt")
	CLUSTERENV := JWTSECRET=testing
	TEST_AUTHENTICATION := jwt:root:
endif
ifeq ("$(TEST_SSL)", "auto")
	CLUSTERENV := SSL=auto $(CLUSTERENV)
	TEST_ENDPOINTS = https://localhost:7001
endif

ifeq ("$(TEST_CONNECTION)", "vst")
	TESTS := $(REPOPATH)/test
ifndef TEST_CONTENT_TYPE
	TEST_CONTENT_TYPE := vpack
endif
endif

ifeq ("$(TEST_BENCHMARK)", "true")
	TAGS := -bench=. -run=notests -cpu=1,2,4
	TESTS := $(REPOPATH)/test
endif

ifdef TEST_ENDPOINTS_OVERRIDE
	TEST_NET := host 
	TEST_ENDPOINTS := $(TEST_ENDPOINTS_OVERRIDE)
endif

ifdef ENABLE_VST11
	VST11_SINGLE_TESTS := run-tests-single-vst-1.1
	VST11_RESILIENTSINGLE_TESTS := run-tests-resilientsingle-vst-1.1
	VST11_CLUSTER_TESTS := run-tests-cluster-vst-1.1
endif

.PHONY: all build clean run-tests

all: build

build: $(GOBUILDDIR) $(SOURCES)
	GOPATH=$(GOBUILDDIR) go build -v $(REPOPATH) $(REPOPATH)/http $(REPOPATH)/vst $(REPOPATH)/agency $(REPOPATH)/jwt

clean:
	rm -Rf $(GOBUILDDIR)

$(GOBUILDDIR):
	@mkdir -p $(ORGDIR)
	@rm -f $(REPODIR) && ln -s ../../../.. $(REPODIR)
	GOPATH=$(GOBUILDDIR) go get github.com/arangodb/go-velocypack
	GOPATH=$(GOBUILDDIR) go get github.com/dgrijalva/jwt-go

.PHONY: changelog
changelog:
	@docker run --rm \
		-e CHANGELOG_GITHUB_TOKEN=$(shell cat ~/.arangodb/github-token) \
		-v "$(ROOTDIR)":/usr/local/src/your-app \
		ferrarimarco/github-changelog-generator \
		--user arangodb \
		--project go-driver \
		--no-author \
		--unreleased-label "Master"

run-tests: run-tests-http run-tests-single run-tests-resilientsingle run-tests-cluster

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
run-tests-single: run-tests-single-json run-tests-single-vpack run-tests-single-vst-1.0 $(VST11_SINGLE_TESTS)

run-tests-single-json: run-tests-single-json-with-auth run-tests-single-json-no-auth run-tests-single-json-ssl

run-tests-single-vpack: run-tests-single-vpack-with-auth run-tests-single-vpack-no-auth run-tests-single-vpack-ssl

run-tests-single-vst-1.0: run-tests-single-vst-1.0-with-auth run-tests-single-vst-1.0-no-auth run-tests-single-vst-1.0-ssl

run-tests-single-vst-1.1: run-tests-single-vst-1.1-with-auth run-tests-single-vst-1.1-jwt-auth run-tests-single-vst-1.1-no-auth run-tests-single-vst-1.1-ssl run-tests-single-vst-1.1-jwt-ssl

run-tests-single-json-no-auth:
	@echo "Single server, HTTP+JSON, no authentication"
	@${MAKE} TEST_MODE="single" TEST_AUTH="none" TEST_CONTENT_TYPE="json" __run_tests

run-tests-single-vpack-no-auth:
	@echo "Single server, HTTP+Velocypack, no authentication"
	@${MAKE} TEST_MODE="single" TEST_AUTH="none" TEST_CONTENT_TYPE="vpack" __run_tests

run-tests-single-vst-1.0-no-auth:
	@echo "Single server, Velocystream 1.0, no authentication"
	@${MAKE} TEST_MODE="single" TEST_AUTH="none" TEST_CONNECTION="vst" TEST_CVERSION="1.0" __run_tests

run-tests-single-vst-1.1-no-auth:
	@echo "Single server, Velocystream 1.1, no authentication"
	@${MAKE} TEST_MODE="single" TEST_AUTH="none" TEST_CONNECTION="vst" TEST_CVERSION="1.1" __run_tests

run-tests-single-json-with-auth:
	@echo "Single server, HTTP+JSON, with authentication"
	@${MAKE} TEST_MODE="single" TEST_AUTH="rootpw" TEST_CONTENT_TYPE="json" __run_tests

run-tests-single-vpack-with-auth:
	@echo "Single server, HTTP+Velocypack, with authentication"
	@${MAKE} TEST_MODE="single" TEST_AUTH="rootpw" TEST_CONTENT_TYPE="vpack" __run_tests

run-tests-single-vst-1.0-with-auth:
	@echo "Single server, Velocystream 1.0, with authentication"
	@${MAKE} TEST_MODE="single" TEST_AUTH="rootpw" TEST_CONNECTION="vst" TEST_CVERSION="1.0" __run_tests

run-tests-single-vst-1.1-with-auth:
	@echo "Single server, Velocystream 1.1, with authentication"
	@${MAKE} TEST_MODE="single" TEST_AUTH="rootpw" TEST_CONNECTION="vst" TEST_CVERSION="1.1" __run_tests

run-tests-single-vst-1.1-jwt-auth:
	@echo "Single server, Velocystream 1.1, JWT authentication"
	@${MAKE} TEST_MODE="single" TEST_AUTH="jwt" TEST_CONNECTION="vst" TEST_CVERSION="1.1" __run_tests

run-tests-single-json-ssl:
	@echo "Single server, HTTP+JSON, with authentication, SSL"
	@${MAKE} TEST_MODE="single" TEST_AUTH="rootpw" TEST_SSL="auto" TEST_CONTENT_TYPE="json" __run_tests

run-tests-single-vpack-ssl:
	@echo "Single server, HTTP+Velocypack, with authentication, SSL"
	@${MAKE} TEST_MODE="single" TEST_AUTH="rootpw" TEST_SSL="auto" TEST_CONTENT_TYPE="vpack" __run_tests

run-tests-single-vst-1.0-ssl:
	@echo "Single server, Velocystream 1.0, with authentication, SSL"
	@${MAKE} TEST_MODE="single" TEST_AUTH="rootpw" TEST_SSL="auto" TEST_CONNECTION="vst" TEST_CVERSION="1.0" __run_tests

run-tests-single-vst-1.1-ssl:
	@echo "Single server, Velocystream 1.1, with authentication, SSL"
	@${MAKE} TEST_MODE="single" TEST_AUTH="rootpw" TEST_SSL="auto" TEST_CONNECTION="vst" TEST_CVERSION="1.1" __run_tests

run-tests-single-vst-1.1-jwt-ssl:
	@echo "Single server, Velocystream 1.1, JWT authentication, SSL"
	@${MAKE} TEST_MODE="single" TEST_AUTH="jwt" TEST_SSL="auto" TEST_CONNECTION="vst" TEST_CVERSION="1.1" __run_tests

# ResilientSingle server tests 
run-tests-resilientsingle: run-tests-resilientsingle-json run-tests-resilientsingle-vpack run-tests-resilientsingle-vst-1.0 $(VST11_RESILIENTSINGLE_TESTS)

run-tests-resilientsingle-json: run-tests-resilientsingle-json-with-auth run-tests-resilientsingle-json-no-auth

run-tests-resilientsingle-vpack: run-tests-resilientsingle-vpack-with-auth run-tests-resilientsingle-vpack-no-auth

run-tests-resilientsingle-vst-1.0: run-tests-resilientsingle-vst-1.0-with-auth run-tests-resilientsingle-vst-1.0-no-auth

run-tests-resilientsingle-vst-1.1: run-tests-resilientsingle-vst-1.1-with-auth run-tests-resilientsingle-vst-1.1-jwt-auth run-tests-resilientsingle-vst-1.1-no-auth

run-tests-resilientsingle-json-no-auth:
	@echo "Resilient Single server, HTTP+JSON, no authentication"
	@${MAKE} TEST_MODE="resilientsingle" TEST_AUTH="none" TEST_CONTENT_TYPE="json" __run_tests

run-tests-resilientsingle-vpack-no-auth:
	@echo "Resilient Single server, HTTP+Velocypack, no authentication"
	@${MAKE} TEST_MODE="resilientsingle" TEST_AUTH="none" TEST_CONTENT_TYPE="vpack" __run_tests

run-tests-resilientsingle-vst-1.0-no-auth:
	@echo "Resilient Single server, Velocystream 1.0, no authentication"
	@${MAKE} TEST_MODE="resilientsingle" TEST_AUTH="none" TEST_CONNECTION="vst" TEST_CVERSION="1.0" __run_tests

run-tests-resilientsingle-vst-1.1-no-auth:
	@echo "Resilient Single server, Velocystream 1.1, no authentication"
	@${MAKE} TEST_MODE="resilientsingle" TEST_AUTH="none" TEST_CONNECTION="vst" TEST_CVERSION="1.1" __run_tests

run-tests-resilientsingle-json-with-auth:
	@echo "Resilient Single server, HTTP+JSON, with authentication"
	@${MAKE} TEST_MODE="resilientsingle" TEST_AUTH="rootpw" TEST_CONTENT_TYPE="json" __run_tests

run-tests-resilientsingle-vpack-with-auth:
	@echo "Resilient Single server, HTTP+Velocypack, with authentication"
	@${MAKE} TEST_MODE="resilientsingle" TEST_AUTH="rootpw" TEST_CONTENT_TYPE="vpack" __run_tests

run-tests-resilientsingle-vst-1.0-with-auth:
	@echo "Resilient Single server, Velocystream 1.0, with authentication"
	@${MAKE} TEST_MODE="resilientsingle" TEST_AUTH="rootpw" TEST_CONNECTION="vst" TEST_CVERSION="1.0" __run_tests

run-tests-resilientsingle-vst-1.1-with-auth:
	@echo "Resilient Single server, Velocystream 1.1, with authentication"
	@${MAKE} TEST_MODE="resilientsingle" TEST_AUTH="rootpw" TEST_CONNECTION="vst" TEST_CVERSION="1.1" __run_tests

run-tests-resilientsingle-vst-1.1-jwt-auth:
	@echo "Resilient Single server, Velocystream 1.1, JWT authentication"
	@${MAKE} TEST_MODE="resilientsingle" TEST_AUTH="jwt" TEST_CONNECTION="vst" TEST_CVERSION="1.1" __run_tests

# Cluster mode tests
run-tests-cluster: run-tests-cluster-json run-tests-cluster-vpack run-tests-cluster-vst-1.0 $(VST11_CLUSTER_TESTS)

run-tests-cluster-json: run-tests-cluster-json-no-auth run-tests-cluster-json-with-auth run-tests-cluster-json-ssl

run-tests-cluster-vpack: run-tests-cluster-vpack-no-auth run-tests-cluster-vpack-with-auth run-tests-cluster-vpack-ssl

run-tests-cluster-vst-1.0: run-tests-cluster-vst-1.0-no-auth run-tests-cluster-vst-1.0-with-auth run-tests-cluster-vst-1.0-ssl

run-tests-cluster-vst-1.1: run-tests-cluster-vst-1.1-no-auth run-tests-cluster-vst-1.1-with-auth run-tests-cluster-vst-1.1-ssl

run-tests-cluster-json-no-auth: $(GOBUILDDIR)
	@echo "Cluster server, JSON, no authentication"
	@${MAKE} TEST_MODE="cluster" TEST_AUTH="none" TEST_CONTENT_TYPE="json" __run_tests

run-tests-cluster-vpack-no-auth: $(GOBUILDDIR)
	@echo "Cluster server, Velocypack, no authentication"
	@${MAKE} TEST_MODE="cluster" TEST_AUTH="none" TEST_CONTENT_TYPE="vpack" __run_tests

run-tests-cluster-vst-1.0-no-auth: $(GOBUILDDIR)
	@echo "Cluster server, Velocystream 1.0, no authentication"
	@${MAKE} TEST_MODE="cluster" TEST_AUTH="none" TEST_CONNECTION="vst" TEST_CVERSION="1.0" __run_tests

run-tests-cluster-vst-1.1-no-auth: $(GOBUILDDIR)
	@echo "Cluster server, Velocystream 1.1, no authentication"
	@${MAKE} TEST_MODE="cluster" TEST_AUTH="none" TEST_CONNECTION="vst" TEST_CVERSION="1.1" __run_tests

run-tests-cluster-json-with-auth: $(GOBUILDDIR)
	@echo "Cluster server, with authentication"
	@${MAKE} TEST_MODE="cluster" TEST_AUTH="rootpw" TEST_CONTENT_TYPE="json" __run_tests

run-tests-cluster-vpack-with-auth: $(GOBUILDDIR)
	@echo "Cluster server, Velocypack, with authentication"
	@${MAKE} TEST_MODE="cluster" TEST_AUTH="rootpw" TEST_CONTENT_TYPE="vpack" __run_tests

run-tests-cluster-vst-1.0-with-auth: $(GOBUILDDIR)
	@echo "Cluster server, Velocystream 1.0, with authentication"
	@${MAKE} TEST_MODE="cluster" TEST_AUTH="rootpw" TEST_CONNECTION="vst" TEST_CVERSION="1.0" __run_tests

run-tests-cluster-vst-1.1-with-auth: $(GOBUILDDIR)
	@echo "Cluster server, Velocystream 1.1, with authentication"
	@${MAKE} TEST_MODE="cluster" TEST_AUTH="rootpw" TEST_CONNECTION="vst" TEST_CVERSION="1.1" __run_tests

run-tests-cluster-json-ssl: $(GOBUILDDIR)
	@echo "Cluster server, SSL, with authentication"
	@${MAKE} TEST_MODE="cluster" TEST_AUTH="rootpw" TEST_SSL="auto" TEST_CONTENT_TYPE="json" __run_tests

run-tests-cluster-vpack-ssl: $(GOBUILDDIR)
	@echo "Cluster server, Velocypack, SSL, with authentication"
	@${MAKE} TEST_MODE="cluster" TEST_AUTH="rootpw" TEST_SSL="auto" TEST_CONTENT_TYPE="vpack" __run_tests

run-tests-cluster-vst-1.0-ssl: $(GOBUILDDIR)
	@echo "Cluster server, Velocystream 1.0, SSL, with authentication"
	@${MAKE} TEST_MODE="cluster" TEST_AUTH="rootpw" TEST_SSL="auto" TEST_CONNECTION="vst" TEST_CVERSION="1.0" __run_tests

run-tests-cluster-vst-1.1-ssl: $(GOBUILDDIR)
	@echo "Cluster server, Velocystream 1.1, SSL, with authentication"
	@${MAKE} TEST_MODE="cluster" TEST_AUTH="rootpw" TEST_SSL="auto" TEST_CONNECTION="vst" TEST_CVERSION="1.1" __run_tests

# Internal test tasks
__run_tests: $(GOBUILDDIR) __test_prepare __test_go_test __test_cleanup

__test_go_test:
	docker run \
		--name=$(TESTCONTAINER) \
		--net=$(TEST_NET) \
		-v $(ROOTDIR):/usr/code \
		-e GOPATH=/usr/code/.gobuild \
		-e GOCACHE=off \
		-e TEST_ENDPOINTS=$(TEST_ENDPOINTS) \
		-e TEST_AUTHENTICATION=$(TEST_AUTHENTICATION) \
		-e TEST_CONNECTION=$(TEST_CONNECTION) \
		-e TEST_CVERSION=$(TEST_CVERSION) \
		-e TEST_CONTENT_TYPE=$(TEST_CONTENT_TYPE) \
		-e TEST_PPROF=$(TEST_PPROF) \
		-e TEST_MODE=$(TEST_MODE) \
		-w /usr/code/ \
		golang:$(GOVERSION) \
		go test $(TAGS) $(TESTOPTIONS) $(TESTVERBOSEOPTIONS) $(TESTS)

__test_prepare:
ifdef TEST_ENDPOINTS_OVERRIDE
	@-docker rm -f -v $(TESTCONTAINER) &> /dev/null
else
ifdef JWTSECRET 
	echo "$JWTSECRET" > "${JWTSECRETFILE}"
endif
	@-docker rm -f -v $(TESTCONTAINER) &> /dev/null
	@TESTCONTAINER=$(TESTCONTAINER) ARANGODB=$(ARANGODB) STARTER=$(STARTER) STARTERMODE=$(TEST_MODE) TMPDIR=${GOBUILDDIR} $(CLUSTERENV) $(ROOTDIR)/test/cluster.sh start
endif

__test_cleanup:
	@docker rm -f -v $(TESTCONTAINER) &> /dev/null
ifndef TEST_ENDPOINTS_OVERRIDE
	@TESTCONTAINER=$(TESTCONTAINER) ARANGODB=$(ARANGODB) STARTER=$(STARTER) STARTERMODE=$(TEST_MODE) $(ROOTDIR)/test/cluster.sh cleanup
endif
	@sleep 3


run-tests-cluster-failover: $(GOBUILDDIR)
	# Note that we use 127.0.0.1:7001.. as endpoints, so we force using IPv4
	# This is essential since we only block IPv4 ports in the test.
	@echo "Cluster server, failover, no authentication"
	@TESTCONTAINER=$(TESTCONTAINER) ARANGODB=$(ARANGODB) $(ROOTDIR)/test/cluster.sh start
	GOPATH=$(GOBUILDDIR) go get github.com/coreos/go-iptables/iptables
	docker run \
		--rm \
		--net=container:$(TESTCONTAINER)-ns \
		--privileged \
		-v $(ROOTDIR):/usr/code \
		-e GOPATH=/usr/code/.gobuild \
		-e TEST_ENDPOINTS=http://127.0.0.1:7001,http://127.0.0.1:7006,http://127.0.0.1:7011 \
		-e TEST_AUTHENTICATION=basic:root: \
		-w /usr/code/ \
		golang:$(GOVERSION) \
		/bin/sh -c 'apk add -U iptables && go test -run ".*Failover.*" -tags failover $(TESTOPTIONS) $(REPOPATH)/test'
	@TESTCONTAINER=$(TESTCONTAINER) ARANGODB=$(ARANGODB) $(ROOTDIR)/test/cluster.sh cleanup

run-tests-cluster-cleanup:
	@TESTCONTAINER=$(TESTCONTAINER) ARANGODB=$(ARANGODB) $(ROOTDIR)/test/cluster.sh cleanup

# Benchmarks
run-benchmarks-single-json-no-auth: 
	@echo "Benchmarks: Single server, JSON no authentication"
	@${MAKE} TEST_MODE="single" TEST_AUTH="none" TEST_CONTENT_TYPE="json" TEST_BENCHMARK="true" __run_tests

run-benchmarks-single-vpack-no-auth: 
	@echo "Benchmarks: Single server, Velocypack, no authentication"
	@${MAKE} TEST_MODE="single" TEST_AUTH="none" TEST_CONTENT_TYPE="vpack" TEST_BENCHMARK="true" __run_tests
