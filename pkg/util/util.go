package util

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/urfave/cli/v2"
	"golang.org/x/sys/unix"
)


// IsTerminal returns true if the file descriptor is terminal.
func IsTerminal(fd uintptr) bool {
	_, err := unix.IoctlGetTermios(int(fd), unix.TCGETS)
	return err == nil
}

// BashCompleteCommand prints all visible flag options for the given command,
// and then recursively calls itself on each subcommand.
func BashCompleteCommand(cmd *cli.Command, w io.Writer) {
	for _, name := range cmd.Names() {
		_, _ = fmt.Fprintf(w, "%v\n", name)
	}

	PrintFlagNames(cmd.VisibleFlags(), w)

	for _, command := range cmd.Subcommands {
		BashCompleteCommand(command, w)
	}
}

// PrintFlagNames prints the long and short names of each flag in the slice.
func PrintFlagNames(flags []cli.Flag, w io.Writer) {
	for _, flag := range flags {
		for _, name := range flag.Names() {
			if len(name) > 1 {
				_, _ = fmt.Fprintf(w, "--%v\n", name)
			} else {
				_, _ = fmt.Fprintf(w, "-%v\n", name)
			}
		}
	}
}

// BashComplete prints all commands, subcommands and flags to the application
// writer.
func BashComplete(c *cli.Context) {
	for _, command := range c.App.VisibleCommands() {
		BashCompleteCommand(command, c.App.Writer)

		// global flags
		PrintFlagNames(c.App.VisibleFlags(), c.App.Writer)
	}
}



// GetLocale tries to get current locale
func GetLocale() string {
	// FIXME: Locale should be detected in more reliable way. We are going to support
	//        localization in better way. Maybe we could use following go module
	//        https://github.com/Xuanwo/go-locale. Maybe some other will be better.
	locale := os.Getenv("LANG")
	return locale
}

// CheckForUnknownArgs returns an error if any unknown arguments are present.
func CheckForUnknownArgs(ctx *cli.Context) error {
	if ctx.Args().Len() != 0 {
		return fmt.Errorf("error: unknown option(s): %s",
			strings.Join(ctx.Args().Slice(), " "))
	}
	return nil
}

