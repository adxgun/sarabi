package cmdutil

import (
	"fmt"
	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"os"
	"time"
)

var (
	loadingSpinner = spinner.New(spinner.CharSets[0], time.Millisecond*100)
)

func PrintE(message string) {
	println()
	color.Red(message)
}

func Print(message string) {
	_, _ = fmt.Fprintln(os.Stdout, message)
}

func PrintS(message string) {
	println()
	color.Green(message)
}

func StartLoading(message string) {
	loadingSpinner.Prefix = message
	loadingSpinner.Start()
}

func StopLoading() {
	loadingSpinner.Stop()
}
