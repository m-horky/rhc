package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/redhatinsights/rhc/internal/datacollection"
	"github.com/redhatinsights/rhc/internal/remotemanagement"
	"github.com/redhatinsights/rhc/internal/rhsm"
	"github.com/redhatinsights/rhc/internal/ui"
	"github.com/redhatinsights/rhc/pkg/feature"
	"github.com/redhatinsights/rhc/pkg/feature/prefcache"
)

type FeatureResult struct {
	Enabled    bool   `json:"enabled"`
	Successful bool   `json:"successful"`
	Error      string `json:"error,omitempty"`
	Skipped    bool   `json:"skipped,omitempty"`
}

// ConnectResult is an external DTO representing the result of 'rhc connect' user action.
type ConnectResult struct {
	Hostname         string `json:"hostname"`
	HostnameError    string `json:"hostname_error,omitempty"`
	UID              int    `json:"uid"`
	UIDError         string `json:"uid_error,omitempty"`
	RHSMConnected    bool   `json:"rhsm_connected"`
	RHSMConnectError string `json:"rhsm_connect_error,omitempty"`
	Features         struct {
		Content          FeatureResult `json:"content"`
		Analytics        FeatureResult `json:"analytics"`
		RemoteManagement FeatureResult `json:"remote_management"`
	} `json:"features"`
	format string
}

// Error implement error interface for structure ConnectResult
func (connectResult *ConnectResult) Error() string {
	var result string
	switch connectResult.format {
	case "json":
		data, err := json.MarshalIndent(connectResult, "", "    ")
		if err != nil {
			return err.Error()
		}
		result = string(data)
	case "":
		break
	default:
		result = "error: unsupported document format: " + connectResult.format
	}
	return result
}

func (connectResult *ConnectResult) errorMessages() map[string]string {
	errorMessages := make(map[string]string)
	if connectResult.RHSMConnectError != "" {
		errorMessages["rhsm"] = connectResult.RHSMConnectError
	}
	if connectResult.Features.Analytics.Error != "" {
		errorMessages["insights"] = connectResult.Features.Analytics.Error
	}
	if connectResult.Features.RemoteManagement.Error != "" {
		errorMessages["yggdrasil"] = connectResult.Features.RemoteManagement.Error
	}
	return errorMessages
}

// TryRegisterRHSM will attempt to register the system with Red Hat Subscription Management.
// If this fails, then both RHSMConnected and Features.Content.Successful will be set to false, and the error message
// will be stored in RHSMConnectError.
func (connectResult *ConnectResult) TryRegisterRHSM(ctx *cli.Context, enableContent bool) {
	slog.Info("Registering the system with Red Hat Subscription Management")
	returnedMsg, err := rhsm.RegisterRHSM(ctx, enableContent)
	if err != nil {
		connectResult.RHSMConnected = false
		connectResult.RHSMConnectError = fmt.Sprintf("cannot connect to Red Hat Subscription Management: %s", err)
		connectResult.Features.Content.Successful = false
		slog.Error(connectResult.RHSMConnectError)
		ui.Printf(
			"%s[%v] Cannot connect to Red Hat Subscription Management\n",
			ui.Indent.Small,
			ui.Icons.Error,
		)
		slog.Warn("Skipping generation of redhat.repo (RHSM registration failed)")
		ui.Printf(
			"%s[%v] Skipping generation of Red Hat repository file\n",
			ui.Indent.Medium,
			ui.Icons.Error,
		)
	} else {
		connectResult.RHSMConnected = true
		slog.Debug(returnedMsg)
		ui.Printf("%s[%v] %s\n", ui.Indent.Small, ui.Icons.Ok, returnedMsg)
		if enableContent {
			connectResult.Features.Content.Successful = true
			slog.Info("redhat.repo has been generated")
			ui.Printf("%s[%v] Content ... Red Hat repository file generated\n", ui.Indent.Medium, ui.Icons.Ok)
		} else {
			connectResult.Features.Content.Successful = false
			slog.Info("redhat.repo not generated (content feature disabled)")
			ui.Printf("%s[%v] Content ... Red Hat repository file absent\n", ui.Indent.Medium, ui.Icons.Info)
		}
	}
}

// TryRegisterInsightsClient will attempt to register the system with Red Hat Lightspeed.
// If this fails, then Features.Analytics.Successful will be set to false, and the
// error message will be stored in Features.Analytics.Error.
func (connectResult *ConnectResult) TryRegisterInsightsClient() {
	slog.Info("Connecting to Red Hat Lightspeed")
	err := ui.Spinner(datacollection.RegisterInsightsClient, ui.Indent.Medium, "Connecting to Red Hat Lightspeed (formerly Insights)...")
	if err != nil {
		connectResult.Features.Analytics.Successful = false
		connectResult.Features.Analytics.Error = fmt.Sprintf("cannot connect to Red Hat Lightspeed: %v", err)
		slog.Error(fmt.Sprintf("cannot connect to Red Hat Lightspeed: %v", err))
		ui.Printf(
			"%s[%v] Analytics ... Cannot connect to Red Hat Lightspeed\n",
			ui.Indent.Medium,
			ui.Icons.Error,
		)
		return
	}

	connectResult.Features.Analytics.Successful = true
	slog.Debug("Connected to Red Hat Lightspeed")
	ui.Printf("%s[%v] Analytics ... Connected to Red Hat Lightspeed (formerly Insights)\n", ui.Indent.Medium, ui.Icons.Ok)
}

// TryEnableYggdrasil will attempt to activate the yggdrasil service.
// If this fails, then Features.RemoteManagement.Successful will be set to false, and the
// error message will be stored in Features.RemoteManagement.Error.
func (connectResult *ConnectResult) TryEnableYggdrasil() {
	slog.Info("Activating yggdrasil service")
	err := ui.Spinner(remotemanagement.ActivateServices, ui.Indent.Medium, " Activating the yggdrasil service")
	if err != nil {
		connectResult.Features.RemoteManagement.Successful = false
		connectResult.Features.RemoteManagement.Error = fmt.Sprintf("cannot activate the yggdrasil service: %v", err)
		slog.Error(connectResult.Features.RemoteManagement.Error)
		ui.Printf(
			"%s[%v] Remote Management ... Cannot activate the yggdrasil service\n",
			ui.Indent.Medium,
			ui.Icons.Error,
		)
		return
	}

	connectResult.Features.RemoteManagement.Successful = true
	infoMsg := "Activated the yggdrasil service"
	slog.Debug(infoMsg)
	ui.Printf("%s[%v] Remote Management ... %s\n", ui.Indent.Medium, ui.Icons.Ok, infoMsg)
}

// checkFeatureFlags validates --enable-feature and --disable-feature flag combinations.
// Returns an error if the combination is invalid.
func checkFeatureFlags(toEnable, toDisable []string) error {
	// Check for feature in both lists
	for _, e := range toEnable {
		for _, d := range toDisable {
			if e == d {
				return fmt.Errorf("invalid combination: enable '%s', disable '%s'", e, d)
			}
		}
	}

	// Check if enabling a feature while disabling its dependencies
	for _, e := range toEnable {
		f, err := feature.Get(e)
		if err != nil {
			return err
		}
		for _, dep := range f.Requires() {
			for _, d := range toDisable {
				if dep == d {
					return fmt.Errorf("invalid combination: enable '%s', disable '%s'", e, d)
				}
			}
		}
	}

	// Check if disabling a feature while enabling features that depend on it
	for _, d := range toDisable {
		f, err := feature.Get(d)
		if err != nil {
			return err
		}
		for _, dependent := range f.RequiredBy() {
			for _, e := range toEnable {
				if dependent == e {
					return fmt.Errorf("invalid combination: enable '%s', disable '%s'", e, d)
				}
			}
		}
	}

	return nil
}

// beforeConnectAction ensures correct CLI flags have been passed in:
// correct values, no conflicts. On error, this method invokes cli.Exit()
// with appropriate message and error code.
func beforeConnectAction(ctx *cli.Context) error {
	// Verify --format contains valid value
	err := checkFormatFlag(ctx)
	if err != nil {
		return err
	}
	// Configure UI globals
	configureUI(ctx)

	// Validate --enable-feature/--disable-feature combinations make sense
	err = checkFeatureFlags(
		ctx.StringSlice("enable-feature"),
		ctx.StringSlice("disable-feature"),
	)
	if err != nil {
		return cli.Exit(err.Error(), ExitCodeUsage)
	}

	// Do not continue if the host is already registered
	slog.Info("Checking system connection status")
	uuid, err := rhsm.GetConsumerUUID()
	if err != nil {
		return cli.Exit(
			fmt.Sprintf("unable to get consumer UUID: %s", err),
			ExitCodeSoftware,
		)
	}
	if uuid != "" {
		slog.Info("Consumer UUID is set, system is already connected")
		return cli.Exit("this system is already connected", ExitCodeUsage)
	}

	username := ctx.String("username")
	password := ctx.String("password")
	organization := ctx.String("organization")
	activationKeys := ctx.StringSlice("activation-key")
	contentTemplates := ctx.StringSlice("content-template")

	if len(activationKeys) > 0 {
		if username != "" {
			exitErr := cli.Exit(
				"--username and --activation-key can not be used together",
				ExitCodeUsage,
			)
			return exitErr

		}
		if password != "" {
			exitErr := cli.Exit(
				"--password and --activation-key can not be used together",
				ExitCodeUsage,
			)
			return exitErr

		}
		if organization == "" {
			exitErr := cli.Exit(
				"--organization is required, when --activation-key is used",
				ExitCodeUsage,
			)
			return exitErr
		}
	}

	// Exit if username/password or activation key/organization haven't been provided,
	// and we cannot ask interactively.
	if !ui.IsInteractive() {
		if (username == "" || password == "") && (len(activationKeys) == 0 || organization == "") {
			exitErr := cli.Exit(
				"--username/--password or --organization/--activation-key are required when a machine-readable format is used",
				ExitCodeUsage,
			)
			return exitErr
		}
	}

	// Load preference cache created by 'rhc configure features'.
	// If missing, it returns default cache.
	cache, err := prefcache.LoadCache(ConnectFeaturesPrefsPath)
	if err != nil {
		return cli.Exit(fmt.Sprintf("error: failed to load preferences: %v", err), ExitCodeSoftware)
	}
	if len(ctx.StringSlice("enable-feature")) > 0 || len(ctx.StringSlice("disable-feature")) > 0 {
		for _, f := range ctx.StringSlice("enable-feature") {
			if err = cache.Set(f, true); err != nil {
				return cli.Exit(fmt.Sprintf("error: %v", err), ExitCodeDataErr)
			}
		}
		for _, f := range ctx.StringSlice("disable-feature") {
			if err = cache.Set(f, false); err != nil {
				return cli.Exit(fmt.Sprintf("error: %v", err), ExitCodeDataErr)
			}
		}
		fmt.Println("Notice: ignoring preferences set via 'rhc configure features'")
		fmt.Println()
	}
	ctx.App.Metadata[ctxConnectCache] = cache

	// Error out if we're trying to set content templates without having enabling content
	contentEnabled, err := cache.Get("content")
	if err != nil {
		return cli.Exit(fmt.Sprintf("error: failed to get content preference: %v", err), ExitCodeSoftware)
	}
	if !contentEnabled && len(contentTemplates) > 0 {
		return cli.Exit("error: content feature is disabled, cannot use --content-template", ExitCodeUsage)
	}

	err = checkForUnknownArgs(ctx)
	if err != nil {
		return cli.Exit(err.Error(), ExitCodeUsage)
	}

	return nil
}

// connectAction manages 'rhc connect' steps:
// first we register against Red Hat Subscription Management,
// then we enable data collection for Red Hat Lightspeed services,
// then we start remote management service yggdrasil.
func connectAction(ctx *cli.Context) error {
	logCommandStart(ctx)
	cache := ctx.App.Metadata[ctxConnectCache].(*prefcache.PreferenceCache)

	// FIXME Rewrite connectResult so the methods aren't mutating it
	var connectResult ConnectResult
	connectResult.format = ctx.String("format")

	uid := os.Getuid()
	if uid != 0 {
		errMsg := "non-root user cannot connect system"
		exitCode := 1
		slog.Error(errMsg)
		if ui.IsOutputMachineReadable() {
			connectResult.UID = uid
			connectResult.UIDError = errMsg
			return cli.Exit(connectResult, exitCode)
		}
		return cli.Exit(fmt.Errorf("error: %s", errMsg), exitCode)
	}

	// Gather hostname
	hostname, err := os.Hostname()
	if err != nil {
		slog.Error(fmt.Sprintf("Error retrieving system hostname: %v", err))
		if ui.IsOutputMachineReadable() {
			connectResult.HostnameError = err.Error()
			return cli.Exit(connectResult, ExitCodeErr)
		}
		return cli.Exit(err, ExitCodeErr)
	}
	connectResult.Hostname = hostname

	ui.Printf("Connecting %v to Red Hat.", hostname)
	var toEnableList []string
	contentEnabled, err := cache.Get("content")
	if err != nil {
		return cli.Exit(fmt.Sprintf("error: failed to get content preference: %v", err), ExitCodeSoftware)
	}
	if contentEnabled {
		toEnableList = append(toEnableList, "content")
	}
	analyticsEnabled, err := cache.Get("analytics")
	if err != nil {
		return cli.Exit(fmt.Sprintf("error: failed to get analytics preference: %v", err), ExitCodeSoftware)
	}
	if analyticsEnabled {
		toEnableList = append(toEnableList, "analytics")
	}
	remoteManagementEnabled, err := cache.Get("remote-management")
	if err != nil {
		return cli.Exit(fmt.Sprintf("error: failed to get remote-management preference: %v", err), ExitCodeSoftware)
	}
	if remoteManagementEnabled {
		toEnableList = append(toEnableList, "remote management")
	}
	if len(toEnableList) > 0 {
		ui.Printf(" ")
		ui.Printf("Enabled features: %s.", strings.Join(toEnableList, ", "))
	}
	ui.Printf("\nThis might take some time.\n\n")

	var start time.Time
	durations := make(map[string]time.Duration)

	// TODO: Refactor to use IFeature interface instead of direct function calls
	// This would make connect consistent with 'rhc configure features' and eliminate
	// duplicate dependency management logic. See configure_features_cmd.go for reference.

	// Register to Red Hat Subscription Management
	{
		start = time.Now()
		contentRequested, err := ctx.App.Metadata[ctxConnectCache].(*prefcache.PreferenceCache).Get("content")
		if err != nil {
			return cli.Exit(fmt.Sprintf("error: failed to get content preference: %v", err), ExitCodeSoftware)
		}
		connectResult.TryRegisterRHSM(
			ctx,
			contentRequested,
		)
		durations["rhsm"] = time.Since(start)
	}

	// Enable data collection
	analyticsRequested, err := ctx.App.Metadata[ctxConnectCache].(*prefcache.PreferenceCache).Get("analytics")
	if err != nil {
		return cli.Exit(fmt.Sprintf("error: failed to get analytics preference: %v", err), ExitCodeSoftware)
	}
	if analyticsRequested {
		start = time.Now()
		connectResult.TryRegisterInsightsClient()
		durations["insights"] = time.Since(start)
	} else {
		ui.Printf("%s[%v] Analytics ... Skipped\n", ui.Indent.Medium, ui.Icons.Info)
	}

	// Enable remote management
	remoteManagementRequested, err := ctx.App.Metadata[ctxConnectCache].(*prefcache.PreferenceCache).Get("remote-management")
	if err != nil {
		return cli.Exit(fmt.Sprintf("error: failed to get remote-management preference: %v", err), ExitCodeSoftware)
	}
	if remoteManagementRequested {
		if !connectResult.Features.Analytics.Successful {
			connectResult.Features.RemoteManagement.Skipped = true
			connectResult.Features.RemoteManagement.Successful = false
			connectResult.Features.RemoteManagement.Error = "skipped: dependency 'analytics' failed"
			slog.Warn("Skipping remote-management (dependency 'analytics' failed)")
			ui.Printf(
				"%s[%v] Remote Management ... Skipped (dependency 'analytics' failed)\n",
				ui.Indent.Medium,
				ui.Icons.Warning,
			)
		} else {
			start = time.Now()
			connectResult.TryEnableYggdrasil()
			durations["yggdrasil"] = time.Since(start)
		}
	} else {
		ui.Printf("%s[%v] Remote Management ... Skipped\n", ui.Indent.Medium, ui.Icons.Info)
	}

	if connectResult.RHSMConnected {
		ui.Printf("\nSuccessfully connected to Red Hat!\n")
	}

	if !ui.IsOutputMachineReadable() {
		/* 5. Show footer message */
		ui.Printf("\nManage your connected systems: https://red.ht/connector\n")

		/* 6. Optionally display duration time of each sub-action */
		showTimeDuration(durations)
	}

	err = showErrorMessages("connect", connectResult.errorMessages())
	if err != nil {
		return err
	}

	if ui.IsOutputMachineReadable() {
		connectResult.Features.Content.Enabled, _ = feature.MustGet("content").IsEnabled()
		connectResult.Features.Analytics.Enabled, _ = feature.MustGet("analytics").IsEnabled()
		connectResult.Features.RemoteManagement.Enabled, _ = feature.MustGet("remote-management").IsEnabled()
		fmt.Println(connectResult.Error())
	}

	err = ctx.App.Metadata[ctxConnectCache].(*prefcache.PreferenceCache).Delete()
	if err != nil {
		slog.Debug("could not delete preferences cache", "err", err)
	}
	return nil
}
