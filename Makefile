all: lint test install

test:
	go test ./...

vet:
	go vet ./...

golint: install-tools
	golint ./...

staticcheck: install-tools
	staticcheck ./...

install-tools:
	cd tools && \
		go list -tags tools -f '{{range $$i := .Imports}}{{printf "%s\n" $$i}}{{end}}' | xargs go install

check-fmt:
	bash -c "diff --line-format='%L' <(echo -n) <(gofmt -d -s .)"

lint: check-fmt vet golint

install:
	go install ./...
