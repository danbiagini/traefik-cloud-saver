.PHONY: lint test vendor clean

export GO111MODULE=on

default: lint test

lint:
	golangci-lint run

test: vendor
	go test -v -cover ./...

yaegi_test:
	yaegi test .

vendor:
	go mod tidy
	go mod vendor

clean:
	rm -rf ./vendor

build-test-container: test
	cd test && docker compose build traefik

# run test container, depend on build-test-container
run-test-container: build-test-container
	cd test && docker compose up -d

stop-test-container:
	cd test && docker compose down
