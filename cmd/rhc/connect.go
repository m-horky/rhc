package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/redhatinsights/rhc/pkg/activation"
	"github.com/redhatinsights/rhc/pkg/config"
	"github.com/redhatinsights/rhc/pkg/features"
	"github.com/redhatinsights/rhc/pkg/insights"
	"github.com/redhatinsights/rhc/pkg/interactive"
	"github.com/redhatinsights/rhc/pkg/rhsm"
)

type FeatureResult struct {
	Enabled    bool   `json:"enabled"`
	Successful bool   `json:"successful"`
	Error      string `json:"error,omitempty"`
}

// ConnectResult is structure holding information about results
// of connect command. The result could be printed in machine-readable format.
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
func (connectResult ConnectResult) Error() string {
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

// String returns string representation of ConnectResult
func (connectResult ConnectResult) String() string {
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

// connectAction tries to connect system to Red Hat Insights and Red Hat Subscription Management
func connectAction(ctx *cli.Context) error {
	uiSettings := interactive.ConfigureUISettings(ctx)

	var connectResult ConnectResult
	durations := make(map[string]time.Duration)

	connectResult.format = ctx.String("format")

	// Collect some basic information about the system.
	start := time.Now()
	hostname, err := os.Hostname()
	stop := time.Now()
	durations["hostname"] = stop.Sub(start)
	if err != nil {
		if uiSettings.IsMachineReadable {
			connectResult.HostnameError = err.Error()
		} else {
			slog.Error("unable to get hostname", "err", err)
		}
	} else {
		connectResult.Hostname = hostname
	}

	start = time.Now()
	uid := os.Getuid()
	stop = time.Now()
	durations["uid"] = stop.Sub(start)
	connectResult.UID = uid

	// When user is not root, then print only warning and continue
	if uid != 0 {
		warningMsg := fmt.Sprintf("not running as root user (UID %v), functionality may be limited", uid)
		if uiSettings.IsMachineReadable {
			connectResult.UIDError = warningMsg
		} else {
			fmt.Printf("%v %v\n", uiSettings.IconInfo, warningMsg)
		}
	}

	// Check feature flags provided on command line
	err = features.CheckFeatureInput(ctx.StringSlice("enable-feature"), ctx.StringSlice("disable-feature"))
	if err != nil {
		return cli.Exit(err, config.ExitCodeDataErr)
	}

	// 1. Register system against Red Hat Subscription Management
	start = time.Now()
	rhsmMsg, err := rhsm.Register(ctx, features.ContentFeature.Enabled, rhsm.UISettings{
		IsRich:            uiSettings.IsRich,
		IsMachineReadable: uiSettings.IsMachineReadable,
		SmallIndent:       interactive.SmallIndent,
	})
	stop = time.Now()
	durations["rhsm-register"] = stop.Sub(start)
	if err != nil {
		if uiSettings.IsMachineReadable {
			connectResult.RHSMConnectError = err.Error()
		} else {
			return err
		}
	} else {
		connectResult.RHSMConnected = true
		if !uiSettings.IsMachineReadable {
			fmt.Printf("%v %v\n", uiSettings.IconOK, rhsmMsg)
		}
	}

	// 2. Register system against Red Hat Insights
	start = time.Now()
	var insightsErr error
	if features.AnalyticsFeature.Enabled {
		insightsErr = insights.Register()
	}
	stop = time.Now()
	durations["insights-register"] = stop.Sub(start)
	if insightsErr != nil {
		if uiSettings.IsMachineReadable {
			connectResult.Features.Analytics.Error = insightsErr.Error()
		} else {
			fmt.Printf("%v Unable to register system to Red Hat Insights: %v\n", uiSettings.IconError, insightsErr)
		}
	} else {
		connectResult.Features.Analytics.Enabled = features.AnalyticsFeature.Enabled
		connectResult.Features.Analytics.Successful = true
		if !uiSettings.IsMachineReadable {
			if features.AnalyticsFeature.Enabled {
				fmt.Printf("%v Connected to Red Hat Insights\n", uiSettings.IconOK)
			} else {
				fmt.Printf("%v Skipping Red Hat Insights registration (%v)\n", uiSettings.IconInfo, features.AnalyticsFeature.Reason)
			}
		}
	}

	// 3. Activate rhc service
	start = time.Now()
	var activateErr error
	if features.ManagementFeature.Enabled {
		activateErr = activation.ActivateService()
	}
	stop = time.Now()
	durations["activate-service"] = stop.Sub(start)
	if activateErr != nil {
		if uiSettings.IsMachineReadable {
			connectResult.Features.RemoteManagement.Error = activateErr.Error()
		} else {
			fmt.Printf("%v Unable to activate %v service: %v\n", uiSettings.IconError, config.ServiceName, activateErr)
		}
	} else {
		connectResult.Features.RemoteManagement.Enabled = features.ManagementFeature.Enabled
		connectResult.Features.RemoteManagement.Successful = true
		if !uiSettings.IsMachineReadable {
			if features.ManagementFeature.Enabled {
				fmt.Printf("%v Activated %v service\n", uiSettings.IconOK, config.ServiceName)
			} else {
				fmt.Printf("%v Skipping activation of %v service (%v)\n", uiSettings.IconInfo, config.ServiceName, features.ManagementFeature.Reason)
			}
		}
	}

	// Set content feature results
	connectResult.Features.Content.Enabled = features.ContentFeature.Enabled
	connectResult.Features.Content.Successful = true

	// Print durations when log level is debug
	interactive.ShowTimeDuration(durations)

	if uiSettings.IsMachineReadable {
		fmt.Print(connectResult.String())
	} else {
		interactive.InteractivePrintf("Manage your connected systems: https://red.ht/connector\n", uiSettings)
	}

	return nil
}