package main

import (
	"os"
	"sync"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"
	"github.com/muesli/termenv"
)

var isInputTerminal = sync.OnceValue(func() bool {
	return isatty.IsTerminal(os.Stdin.Fd())
})

var isOutputTerminal = sync.OnceValue(func() bool {
	return isatty.IsTerminal(os.Stdout.Fd())
})

var stderrRenderer = sync.OnceValue(
	func() *lipgloss.Renderer {
		return lipgloss.NewRenderer(os.Stderr, termenv.WithColorCache(true))
	})

var stderrStyles = sync.OnceValue(func() styles {
	return makeStyles(stderrRenderer())
})
