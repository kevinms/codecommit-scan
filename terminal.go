package main

import (
	"fmt"
	"os"
	"sync"

	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
)

var DebugMode bool

var singleLineMode bool = true
var modeMutex sync.Mutex

type OnDisableBehavior int

const (
	OnDisableNewLine   OnDisableBehavior = 1
	OnDisableClearLine OnDisableBehavior = 2
)

// var ctag = color.New(color.Bold, color.FgRed).SprintFunc()
// var ctext = color.New(color.Bold, color.FgYellow).SprintFunc()
var ctag = color.New(color.FgCyan).SprintFunc()
var ctext = color.New(color.FgYellow).SprintFunc()

func isTerminal(f *os.File) bool {
	return isatty.IsTerminal(f.Fd()) || isatty.IsCygwinTerminal(f.Fd())
}

func clearLine(f *os.File) {
	if !isTerminal(f) {
		return
	}

	fmt.Fprint(f, "\n\033[1A\033[K")
}

// DisableSingleLineMode will ensure all future messages will not overwrite
// previous ones.
func DisableSingleLineMode(behavior OnDisableBehavior) {
	modeMutex.Lock()
	defer modeMutex.Unlock()

	if !singleLineMode {
		return
	}

	// Exiting single line mode, so clear the line one last time.
	singleLineMode = false
	if behavior == OnDisableClearLine {
		clearLine(os.Stderr)
	} else if behavior == OnDisableNewLine {
		fmt.Fprintln(os.Stderr)
	}
}

// _println is an internal function for printing.
//
// In singleLineMode, a newline character is not printed.
func _println(f *os.File, prefix string, a ...interface{}) {
	modeMutex.Lock()
	defer modeMutex.Unlock()

	if singleLineMode {
		clearLine(f)
	}

	fmt.Fprint(f, prefix)
	fmt.Fprint(f, a...)

	if !singleLineMode {
		fmt.Fprintln(f)
	}
}

// Println goes to stdout.
//
// Calling Println always disables single line mode.
func Println(a ...interface{}) {
	// Ensure single line mode is off.
	DisableSingleLineMode(OnDisableClearLine)

	_println(os.Stdout, "", a...)
}

// Infoln goes to stderr.
//
// In single line mode, INFO messages will keep rewriting the same line.
func Infoln(a ...interface{}) {
	_println(os.Stderr, "[INFO]: ", a...)
}

// Debugln goes to stderr.
//
// Calling Debugln always disables single line mode. Debug messages are most
// useful when you can see them all.
func Debugln(a ...interface{}) {
	if !DebugMode {
		return
	}

	// Ensure single line mode is off.
	DisableSingleLineMode(OnDisableNewLine)

	_println(os.Stderr, "[DEBUG]: ", a...)
}

// Fatalln goes to stderr.
//
// Calling Fatalln always disables single line mode. Fatal messages are most
// useful when you can see what immediatly preceeded it.
func Fatalln(a ...interface{}) {
	// Ensure single line mode is off.
	DisableSingleLineMode(OnDisableNewLine)

	_println(os.Stderr, "[FATAL]: ", a...)
	os.Exit(1)
}

func init() {
	if !isTerminal(os.Stderr) {
		color.NoColor = true
		DisableSingleLineMode(OnDisableNewLine)
	}
}
