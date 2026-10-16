//go:build tools

// Package tools pins golang.org/x/mobile, which `gomobile bind` needs in this
// module's dependency graph.
package tools

import _ "golang.org/x/mobile/bind"
