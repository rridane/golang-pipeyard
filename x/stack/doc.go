// Package stack holds what the ready-made stages share: the files a check has
// to look at, and how a pinned tool is run over them. Everything hard lives
// here, so a stage stays short enough to read, and to own once copied.
package stack
