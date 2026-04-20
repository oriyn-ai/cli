package cmd

import (
	"io"
	"os"
)

// stdinReader exists so tests can swap stdin without touching os.Stdin.
var stdinReader io.Reader = os.Stdin
