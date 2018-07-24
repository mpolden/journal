all: deps lint test install

fmt:
	go fmt ./...

test:
	go test ./...

vet:
	go vet ./...

megacheck:
ifdef TRAVIS
	megacheck 2> /dev/null; if [ $$? -eq 127 ]; then \
		go get -v honnef.co/go/tools/cmd/megacheck; \
	fi
	megacheck ./...
endif

golint:
ifdef TRAVIS
	golint 2> /dev/null; if [ $$? -eq 127 ]; then \
		go get -v github.com/golang/lint/golint; \
	fi
	golint ./...
endif

check-fmt:
	bash -c "diff --line-format='%L' <(echo -n) <(gofmt -d -s .)"

lint: check-fmt vet megacheck golint

deps:
	go get -d -v ./...

install:
	go install ./...
