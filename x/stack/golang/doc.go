// Package golang holds the stages a Go repository wants: that it compiles, that
// it is formatted, that vet finds nothing, that its tests pass, and that the
// linter agrees.
//
// Four of the five run on the host, with the toolchain the runner installed.
// That is deliberate: the race detector needs cgo, and asking every CI image
// to carry a C compiler to run `go test -race` costs more than it gives. Only
// the linter runs in a container, because its version is worth pinning and it
// needs no toolchain of its own.
package golang
