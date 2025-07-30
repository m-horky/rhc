package main

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"

	"github.com/redhatinsights/rhc/pkg/config"
	"github.com/redhatinsights/rhc/pkg/features"
	"github.com/redhatinsights/rhc/pkg/interactive"
	"github.com/redhatinsights/rhc/pkg/util"
)

// mainAction is triggered in the case, when no sub-command is specified
func mainAction(c *cli.Context) error {
	type GenerationFunc func() (string, error)
	var generationFunc GenerationFunc
	if c.Bool("generate-man-page") {
		generationFunc = c.App.ToMan
	} else if c.Bool("generate-markdown") {
		generationFunc = c.App.ToMarkdown
	} else {
		cli.ShowAppHelpAndExit(c, 0)
	}
	data, err := generationFunc()
	if err != nil {
		return cli.Exit(err, 1)
	}
	fmt.Println(data)
	return nil
}

// beforeAction is triggered before other actions are triggered
func beforeAction(c *cli.Context) error {
	// check if --log-level was set via command line
	var logLevelSrc string
	if c.IsSet(config.CliLogLevel) {
		logLevelSrc = "command line"
	}

	/* Load the configuration values from the config file specified*/
	filePath := c.String("config")
	if filePath != "" {
		inputSource, err := altsrc.NewTomlSourceFromFile(filePath)
		if err != nil {
			return err
		}
		if err := altsrc.ApplyInputSourceValues(c, inputSource, c.App.Flags); err != nil {
			return err
		}
	}

	// check if log-level was set via config file (command line has precedence)
	if logLevelSrc == "" && c.IsSet(config.CliLogLevel) {
		logLevelSrc = fmt.Sprintf("config file: '%s'", c.String("config"))
	}

	config.Global = config.Config{
		CertFile: c.String(config.CliCertFile),
		KeyFile:  c.String(config.CliKeyFile),
	}

	logLevelStr := c.String(config.CliLogLevel)
	if err := config.Global.LogLevel.UnmarshalText([]byte(logLevelStr)); err != nil {
		slog.Error(fmt.Sprintf("invalid log level '%s' set via %s", logLevelStr, logLevelSrc))
		config.Global.LogLevel = slog.LevelInfo
	}

	slog.SetLogLoggerLevel(config.Global.LogLevel)

	// When environment variable NO_COLOR or --no-color CLI option is set, then do not display colors
	// and animations too. The NO_COLOR environment variable have to have value "1" or "true",
	// "True", "TRUE" to take effect
	// When no-color is not set, then try to detect if the output goes to some file. In this case
	// colors nor animations will not be printed to file.
	if !util.IsTerminal(os.Stdout.Fd()) {
		err := c.Set("no-color", "true")
		if err != nil {
			slog.Debug("Unable to set no-color flag to \"true\"")
		}
	}

	return nil
}

func main() {
	app := cli.NewApp()
	app.Name = config.ShortName
	app.Version = config.Version
	app.Usage = "control the system's connection to " + config.Provider
	app.Description = "The " + app.Name + " command controls the system's connection to " + config.Provider + ".\n\n" +
		"To connect the system using an activation key:\n" +
		"\t" + app.Name + " connect --organization ID --activation-key KEY\n\n" +
		"To connect the system using a username and password:\n" +
		"\t" + app.Name + " connect --username USERNAME --password PASSWORD\n\n" +
		"To disconnect the system:\n" +
		"\t" + app.Name + " disconnect\n\n" +
		"Run '" + app.Name + " command --help' for more details."

	var featureIdSlice []string
	for _, featureID := range features.KnownFeatures {
		featureIdSlice = append(featureIdSlice, featureID.ID)
	}
	featureIDs := strings.Join(featureIdSlice, ", ")

	defaultConfigFilePath, err := config.ConfigPath()
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	app.Flags = []cli.Flag{
		&cli.BoolFlag{
			Name:   "generate-man-page",
			Hidden: true,
		},
		&cli.BoolFlag{
			Name:   "generate-markdown",
			Hidden: true,
		},
		&cli.BoolFlag{
			Name:    "no-color",
			Hidden:  false,
			Value:   false,
			EnvVars: []string{"NO_COLOR"},
		},
		&cli.StringFlag{
			Name:      "config",
			Hidden:    true,
			Value:     defaultConfigFilePath,
			TakesFile: true,
			Usage:     "Read config values from `FILE`",
		},
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:   config.CliCertFile,
			Hidden: true,
			Usage:  "Use `FILE` as the client certificate",
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:   config.CliKeyFile,
			Hidden: true,
			Usage:  "Use `FILE` as the client's private key",
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:   config.CliLogLevel,
			Value:  "info",
			Hidden: true,
			Usage:  "Set the logging output level to `LEVEL`",
		}),
	}

	app.Commands = []*cli.Command{
		{
			Name: "connect",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "username",
					Usage:   "register with `USERNAME`",
					Aliases: []string{"u"},
				},
				&cli.StringFlag{
					Name:    "password",
					Usage:   "register with `PASSWORD`",
					Aliases: []string{"p"},
				},
				&cli.StringFlag{
					Name:    "organization",
					Usage:   "register with `ID`",
					Aliases: []string{"o"},
				},
				&cli.StringSliceFlag{
					Name:    "activation-key",
					Usage:   "register with `KEY`",
					Aliases: []string{"a"},
				},
				&cli.StringSliceFlag{
					Name:    "content-template",
					Usage:   "register with `CONTENT_TEMPLATE`",
					Aliases: []string{"c"},
				},
				&cli.StringSliceFlag{
					Name:    "enable-feature",
					Usage:   fmt.Sprintf("enable `FEATURE` during connection (allowed values: %s)", featureIDs),
					Aliases: []string{"e"},
				},
				&cli.StringSliceFlag{
					Name:    "disable-feature",
					Usage:   fmt.Sprintf("disable `FEATURE` during connection (allowed values: %s)", featureIDs),
					Aliases: []string{"d"},
				},
				&cli.StringFlag{
					Name:    "format",
					Usage:   "prints output of connection in machine-readable format (supported formats: \"json\")",
					Aliases: []string{"f"},
				},
			},
			Usage:       "Connects the system to " + config.Provider,
			UsageText:   app.Name + " connect [command options]",
			Description: "The connect command connects the system to " + config.Provider + " and enables system management.",
			BashComplete: func(c *cli.Context) {
				util.BashComplete(c)
			},
			Before: func(c *cli.Context) error {
				err := util.CheckForUnknownArgs(c)
				if err != nil {
					return err
				}
				uiSettings := interactive.ConfigureUISettings(c)
				return interactive.SetupFormatOption(c, &uiSettings, config.ExitCodeDataErr)
			},
			Action: connectAction,
		},
		{
			Name: "disconnect",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "format",
					Usage:   "prints output of disconnection in machine-readable format (supported formats: \"json\")",
					Aliases: []string{"f"},
				},
			},
			Usage:       "Disconnects the system from " + config.Provider,
			UsageText:   app.Name + " disconnect [command options]",
			Description: "The disconnect command disconnects the system from " + config.Provider + " and disables system management.",
			BashComplete: func(c *cli.Context) {
				util.BashComplete(c)
			},
			Before: func(c *cli.Context) error {
				err := util.CheckForUnknownArgs(c)
				if err != nil {
					return err
				}
				uiSettings := interactive.ConfigureUISettings(c)
				return interactive.SetupFormatOption(c, &uiSettings, config.ExitCodeDataErr)
			},
			Action: disconnectAction,
		},
		{
			Name: "status",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "format",
					Usage:   "prints output of status in machine-readable format (supported formats: \"json\")",
					Aliases: []string{"f"},
				},
			},
			Usage:       "Prints status of the system's connection to " + config.Provider,
			UsageText:   app.Name + " status [command options]",
			Description: "The status command prints the state of the connection to " + config.Provider + ".",
			BashComplete: func(c *cli.Context) {
				util.BashComplete(c)
			},
			Before: func(c *cli.Context) error {
				err := util.CheckForUnknownArgs(c)
				if err != nil {
					return err
				}
				uiSettings := interactive.ConfigureUISettings(c)
				return interactive.SetupFormatOption(c, &uiSettings, config.ExitCodeDataErr)
			},
			Action: statusAction,
		},
		{
			Name:   "canonical-facts",
			Usage:  "Prints canonical facts about the system",
			Hidden: true,
			Action: canonicalFactsAction,
		},
		{
			Name:   "collector",
			Usage:  "Runs the canonical facts collection",
			Hidden: true,
			Action: collectorAction,
		},
	}

	app.Action = mainAction
	app.Before = beforeAction

	app.EnableBashCompletion = true
	app.BashComplete = func(c *cli.Context) {
		util.BashComplete(c)
	}

	if err := app.Run(os.Args); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}