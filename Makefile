MAKE_VARIABLES := $(.VARIABLES)

GO_VERSION=$(shell go version | cut -d ' ' -f 3 | sed -e 's/ /-/g' | sed -e 's/\//-/g' | sed -e 's/^go//g')
GO_PLATFORM ?= $(shell go version | cut -d ' ' -f 4 | sed -e 's/ /-/g' | sed -e 's/\//-/g')
BEATS_VERSION ?= "6.3.0"
BEATS_TAG ?= $(shell echo ${BEATS_VERSION} | sed 's/[^[:digit:]]*\([[:digit:]]*\(\.[[:digit:]]*\)\)/v\1/')
AWSBEATS_VERSION ?= $(shell script/version)
BEAT_NAME ?= "filebeat"
DOCKER_IMAGE ?= s12v/awsbeats
DOCKER_TAG ?= $(AWSBEATS_VERSION)-$(BEAT_NAME)-$(BEATS_VERSION)
BEAT_GITHUB_REPO ?= github.com/elastic/beats
BEAT_GO_PKG ?= $(BEAT_GITHUB_REPO)/$(BEAT_NAME)
BEAT_DOCKER_IMAGE ?= docker.elastic.co/beats/$(BEAT_NAME):$(BEATS_VERSION)
GOPATH ?= $(HOME)/go

.PHONY: all
all: vars test beats build

.PHONY: vars
vars:
	$(info make variables for this build:)
	$(foreach v, $(filter-out $(MAKE_VARIABLES) MAKE_VARIABLES,$(.VARIABLES)), $(info $(v) = $($(v))))
	$(info To override, run make like '`make SOME_VAR=some_value`'.)

test:
	test -z "$$(find . -path ./vendor -prune -type f -o -name '*.go' -exec gofmt -d {} + | tee /dev/stderr)"
	go test ./firehose -v -coverprofile=coverage.txt -covermode=atomic
	go test ./streams -v -coverprofile=coverage.txt -covermode=atomic

format:
	test -z "$$(find . -path ./vendor -prune -type f -o -name '*.go' -exec gofmt -d {} + | tee /dev/stderr)" || \
	test -z "$$(find . -path ./vendor -prune -type f -o -name '*.go' -exec gofmt -w {} + | tee /dev/stderr)"

build: format
	@echo "Building the plugin with: BEATS_VERSION=$(BEATS_VERSION) GO_VERSION=$(GO_VERSION) GO_PLATFORM=$(GO_PLATFORM)"
	@cd "$$GOPATH/src/github.com/elastic/beats" && \
	git checkout $(BEATS_TAG) && \
	cd "$(CURDIR)"
	go build -buildmode=plugin ./plugins/kinesis
	@mkdir -p "$(CURDIR)/target"
	@mv kinesis.so "$(CURDIR)/target/kinesis-$(AWSBEATS_VERSION)-$(BEATS_VERSION)-go$(GO_VERSION)-$(GO_PLATFORM).so"
	@find "$(CURDIR)"/target/

beats: $(GOPATH)/src/$(BEAT_GITHUB_REPO) $(GOPATH)/src/github.com/elastic/beats
ifdef BEATS_VERSION
	@echo "Building $(BEAT_NAME):$(BEATS_VERSION)..."
	@mkdir -p "$(CURDIR)/target"
	@cd "$$GOPATH/src/$(BEAT_GO_PKG)" &&\
	git checkout $(BEATS_TAG)
ifneq ($(GOPATH)/src/$(BEAT_GITHUB_REPO),$(GOPATH)/src/github.com/elastic/beats)
	@echo "Checking out beats $(BEATS_VERSION)"
	@cd "$$GOPATH/src/github.com/elastic/beats" &&\
	git checkout $(BEATS_TAG)
	@echo "Removing vendored libbeat to avoid `flag redefined` errors for the beat outside of the elsatic/beats repo"
	rm -rf "$$GOPATH/src/$(BEAT_GO_PKG)/vendor/github.com/elastic/beats"
	# Work-around for the following build error:
	# cmd/root.go:21:72: cannot use runFlags (type *"github.com/elastic/apm-server/vendor/github.com/spf13/pflag".FlagSet) as type *"github.com/elastic/beats/vendor/github.com/spf13/pflag".FlagSet in argument to cmd.GenRootCmdWithIndexPrefixWithRunFlags
	rm -rf "$$GOPATH/src/$(BEAT_GO_PKG)/vendor/github.com/spf13/pflag"
	rm -rf "$$GOPATH/src/github.com/elastic/beats/vendor/github.com/spf13/pflag"
	go get github.com/spf13/pflag
endif
	@cd "$$GOPATH/src/$(BEAT_GO_PKG)" &&\
	make &&\
	mv $(BEAT_NAME) "$(CURDIR)/target/$(BEAT_NAME)-$(BEATS_VERSION)-go$(GO_VERSION)-$(GO_PLATFORM)"
else
	$(error BEATS_VERSION is undefined)
endif

$(GOPATH)/src/$(BEAT_GITHUB_REPO):
	go get $(BEAT_GITHUB_REPO) || true

ifneq ($(GOPATH)/src/$(BEAT_GITHUB_REPO),$(GOPATH)/src/github.com/elastic/beats)
$(GOPATH)/src/github.com/elastic/beats:
	go get github.com/elastic/beats || true
endif

.PHONY: dockerimage
dockerimage:
	docker build --build-arg AWSBEATS_VERSION=$(AWSBEATS_VERSION) --build-arg GO_VERSION=$(GO_VERSION) --build-arg BEAT_GITHUB_REPO=$(BEAT_GITHUB_REPO) --build-arg BEAT_GO_PKG=$(BEAT_GO_PKG) --build-arg BEAT_DOCKER_IMAGE=$(BEAT_DOCKER_IMAGE) --build-arg BEATS_VERSION=$(BEATS_VERSION) --build-arg BEAT_NAME=$(BEAT_NAME) -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

.PHONY: auditbeat-image
auditbeat-image:
	bash -c 'make dockerimage BEATS_VERSION=$(BEATS_VERSION) GO_VERSION=1.10.2 BEAT_NAME=auditbeat AWSBEATS_VERSION=\$(ref=$(git rev-parse HEAD); ref=${ref:0:7}; echo $ref) GOPATH=$HOME/go'

.PHONY: filebeat-image
filebeat-image:
	bash -c 'make dockerimage BEATS_VERSION=$(BEATS_VERSION) GO_VERSION=1.10.2 BEAT_NAME=filebeat AWSBEATS_VERSION=$(AWSBEATS_VERSION) GOPATH=$HOME/go'

.PHONY: heartbeat-image
heartbeat-image:
	bash -c 'make dockerimage BEATS_VERSION=$(BEATS_VERSION) GO_VERSION=1.10.2 BEAT_NAME=heartbeat AWSBEATS_VERSION=\$(ref=$(git rev-parse HEAD); ref=${ref:0:7}; echo $ref) GOPATH=$HOME/go'

.PHONY: metricbeat-image
metricbeat-image:
	bash -c 'make dockerimage BEATS_VERSION=$(BEATS_VERSION) GO_VERSION=1.10.2 BEAT_NAME=metricbeat AWSBEATS_VERSION=\$(ref=$(git rev-parse HEAD); ref=${ref:0:7}; echo $ref) GOPATH=$HOME/go'

.PHONY: apm-server-image
apm-server-image:
	bash -c 'make dockerimage BEATS_VERSION=$(BEATS_VERSION) GO_VERSION=1.10.2 BEAT_NAME=apm-server BEAT_GITHUB_REPO=github.com/elastic/apm-server BEAT_GO_PKG=github.com/elastic/apm-server BEAT_DOCKER_IMAGE=docker.elastic.co/apm/apm-server:$(BEATS_VERSION) AWSBEATS_VERSION=\$(ref=$(git rev-parse HEAD); ref=${ref:0:7}; echo $ref) GOPATH=$HOME/go'
