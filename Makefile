all: lint test install

deps:
	go get ./...

test: deps
	go test ./...

vet: deps
	go vet ./...

golint: deps
ifdef TRAVIS
	golint 2> /dev/null; if [ $$? -eq 127 ]; then \
		GO111MODULE=off go get -v golang.org/x/lint/golint; \
	fi
	golint ./...
endif

check-fmt:
	bash -c "diff --line-format='%L' <(echo -n) <(gofmt -d -s .)"

lint: check-fmt vet golint

install: deps
	go install ./...
