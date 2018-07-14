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

check-fmt:
	bash -c "diff --line-format='%L' <(echo -n) <(gofmt -d -s .)"

lint: check-fmt vet megacheck

deps:
	go get -d -v ./...

install:
	go install ./...
