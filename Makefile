.PHONY: vendor

SHELL=/bin/bash # Use bash syntax
USER_ID := $(shell id -u)
USER_NAME := $(shell whoami)
GROUP_ID := $(shell id -g)
GROUP_NAME := $(shell id -gn)

vendor:
	go mod vendor

lint:
	golangci-lint run --modules-download-mode vendor -v --max-same-issues 10

install:
	go install -v ./cmd/...

localstack-tarball:
	docker pull localstack/localstack:3.4.0
	docker save -o localstack_3.4.0.tar localstack/localstack:3.4.0
	gzip localstack_3.4.0.tar

dev-container:
	docker build --tag "lomorage/lomo-backup:build-stage0" -f dockerfiles/dev-image .
	docker build --tag lomorage/lomo-backup:1.0 -f dockerfiles/dev-image-merge .


dev:
	docker build \
	    --build-arg USER_ID=$(USER_ID) --build-arg USER_NAME=$(USER_NAME) \
	    --build-arg GROUP_ID=$(GROUP_ID) --build-arg GROUP_NAME=$(GROUP_NAME) \
		--tag "lomorage/lomo-backup" -f dockerfiles/dev-run .
	docker rm -f lomo-backup
	docker run --rm \
		--name lomo-backup --hostname lomo-backup -v ./dockerfiles/hosts:/etc/hosts \
		--privileged --net host --dns-search local \
		-v "${PWD}:/go/src/github.com/lomorage/lomo-backup" \
		-it "lomorage/lomo-backup" -d bash

unit-tests:
	go list ./common/... | xargs -I {} go test -v {}

e2e-tests: install
	cd test/scripts; ./e2e.sh
