// +build tools

// Disable SA5008 because cmd packages has a duplicate "choice" tag
//go:generate go run honnef.co/go/tools/cmd/staticcheck -checks inherit,-SA5008 ./...
//go:generate go run golang.org/x/lint/golint ./...

package tools

import (
	_ "golang.org/x/lint/golint"
	_ "honnef.co/go/tools/cmd/staticcheck"
)
