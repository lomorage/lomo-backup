.PHONY: vendor

SHELL=/bin/bash # Use bash syntax
GOPATH=/go

vendor:
	go mod vendor

lint:
	golangci-lint run --modules-download-mode vendor -v --max-same-issues 10

install:
	go install -v ./cmd/...

dev-container:
	docker build --tag "lomorage/lomo-backup:1.0" -f dockerfiles/dev-image .

dev:
	docker build --tag "lomorage/lomo-backup" -f dockerfiles/dev-run .
	docker rm -f lomo-backup
	docker run \
		--name lomo-backup --hostname lomo-backup \
		--privileged --cap-add=ALL -v /dev:/dev -v /lib/modules:/lib/modules \
		-v "${PWD}:/go/src/github.com/lomorage/lomo-backup" \
		--net host --dns-search local \
		-it "lomorage/lomo-backup" -d bash
