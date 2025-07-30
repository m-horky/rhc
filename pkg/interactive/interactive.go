package interactive

import (
	"fmt"
	"log/slog"
	"os"
	"text/tabwriter"
	"time"

	"github.com/briandowns/spinner"
	"github.com/urfave/cli/v2"

	"github.com/redhatinsights/rhc/pkg/config"
	"github.com/redhatinsights/rhc/pkg/logging"
)

const (
	ColorGreen  = "\u001B[32m"
	ColorYellow = "\u001B[33m"
	ColorRed    = "\u001B[31m"
	ColorReset  = "\u001B[0m"
)

const SmallIndent = " "
const MediumIndent = "  "

// UserInterfaceSettings manages standard output preference.
// It tracks colors, icons and machine-readable output (e.g. json).
type UserInterfaceSettings struct {
	// IsMachineReadable describes the machine-readable mode (e.g., `--format json`)
	IsMachineReadable bool
	// IsRich describes the ability to display colors and animations
	IsRich    bool
	IconOK    string
	IconInfo  string
	IconError string
}

const SymbolOK string = "✓"
const SymbolInfo string = "●"
const SymbolError string = "𐄂"

// ConfigureUISettings is called by the CLI library when it loads up.
// It sets up the uiSettings object.
func ConfigureUISettings(ctx *cli.Context) UserInterfaceSettings {
	if ctx.Bool("no-color") {
		return UserInterfaceSettings{
			IsRich:            false,
			IsMachineReadable: false,
			IconOK:            SymbolOK,
			IconInfo:          SymbolInfo,
			IconError:         SymbolError,
		}
	} else {
		return UserInterfaceSettings{
			IsRich:            true,
			IsMachineReadable: false,
			IconOK:            ColorGreen + SymbolOK + ColorReset,
			IconInfo:          ColorYellow + SymbolInfo + ColorReset,
			IconError:         ColorRed + SymbolError + ColorReset,
		}
	}
}

// ShowProgress calls function and, when it is possible display spinner with
// some progress message.
func ShowProgress(
	progressMessage string,
	function func() error,
	prefixSpaces string,
	uiSettings UserInterfaceSettings,
) error {
	var s *spinner.Spinner
	if uiSettings.IsRich {
		s = spinner.New(spinner.CharSets[9], 100*time.Millisecond)
		s.Prefix = prefixSpaces + "["
		s.Suffix = "]" + progressMessage
		s.Start()
		// Stop spinner after running function
		defer func() { s.Stop() }()
	}
	return function()
}

// ShowTimeDuration shows table with duration of each sub-action
func ShowTimeDuration(durations map[string]time.Duration) {
	if config.Global.LogLevel <= slog.LevelDebug {
		fmt.Println()
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(w, "STEP\tDURATION\t")
		for step, duration := range durations {
			_, _ = fmt.Fprintf(w, "%v\t%v\t\n", step, duration.Truncate(time.Millisecond))
		}
		_ = w.Flush()
	}
}

// ShowErrorMessages shows table with all error messages gathered during action
func ShowErrorMessages(action string, errorMessages map[string]logging.LogMessage, uiSettings UserInterfaceSettings) error {
	if logging.HasPriorityErrors(errorMessages, config.Global.LogLevel) {
		if !uiSettings.IsMachineReadable {
			fmt.Println()
			fmt.Printf("The following errors were encountered during %s:\n\n", action)
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			_, _ = fmt.Fprintln(w, "TYPE\tSTEP\tERROR\t")
			for step, logMsg := range errorMessages {
				if logMsg.Level >= config.Global.LogLevel {
					_, _ = fmt.Fprintf(w, "%v\t%v\t%v\n", logMsg.Level, step, logMsg.Message)
				}
			}
			_ = w.Flush()
			if logging.HasPriorityErrors(errorMessages, slog.LevelError) {
				return cli.Exit("", 1)
			}
		}
	}
	return nil
}

// InteractivePrintf is method for printing human-readable output. It suppresses output, when
// machine-readable format is used.
func InteractivePrintf(format string, uiSettings UserInterfaceSettings, a ...interface{}) {
	if !uiSettings.IsMachineReadable {
		fmt.Printf(format, a...)
	}
}

// SetupFormatOption ensures the user has supplied a correct `--format` flag
// and set values in uiSettings, when JSON format is used.
func SetupFormatOption(ctx *cli.Context, uiSettings *UserInterfaceSettings, exitCodeDataErr int) error {
	// This is run after the `app.Before()` has been run,
	// the uiSettings is already set up for us to modify.
	format := ctx.String("format")
	switch format {
	case "":
		return nil
	case "json":
		uiSettings.IsMachineReadable = true
		uiSettings.IsRich = false
		return nil
	default:
		err := fmt.Errorf(
			"unsupported format: %s (supported formats: %s)",
			format,
			`"json"`,
		)
		return cli.Exit(err, exitCodeDataErr)
	}
}