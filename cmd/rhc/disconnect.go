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
	"github.com/redhatinsights/rhc/pkg/insights"
	"github.com/redhatinsights/rhc/pkg/interactive"
	"github.com/redhatinsights/rhc/pkg/logging"
	"github.com/redhatinsights/rhc/pkg/rhsm"
)

// DisconnectResult is structure holding information about result of
// disconnect command. The result could be printed in machine-readable format.
type DisconnectResult struct {
	Hostname                  string `json:"hostname"`
	HostnameError             string `json:"hostname_error,omitempty"`
	UID                       int    `json:"uid"`
	UIDError                  string `json:"uid_error,omitempty"`
	RHSMDisconnected          bool   `json:"rhsm_disconnected"`
	RHSMDisconnectedError     string `json:"rhsm_disconnect_error,omitempty"`
	InsightsDisconnected      bool   `json:"insights_disconnected"`
	InsightsDisconnectedError string `json:"insights_disconnected_error,omitempty"`
	YggdrasilStopped          bool   `json:"yggdrasil_stopped"`
	YggdrasilStoppedError     string `json:"yggdrasil_stopped_error,omitempty"`
	format                    string
}

// Error implement error interface for structure DisconnectResult
func (disconnectResult DisconnectResult) Error() string {
	var result string
	switch disconnectResult.format {
	case "json":
		data, err := json.MarshalIndent(disconnectResult, "", "    ")
		if err != nil {
			return err.Error()
		}
		result = string(data)
	case "":
		break
	default:
		result = "error: unsupported document format: " + disconnectResult.format
	}
	return result
}

// String returns string representation of DisconnectResult
func (disconnectResult DisconnectResult) String() string {
	var result string
	switch disconnectResult.format {
	case "json":
		data, err := json.MarshalIndent(disconnectResult, "", "    ")
		if err != nil {
			return err.Error()
		}
		result = string(data)
	case "":
		break
	default:
		result = "error: unsupported document format: " + disconnectResult.format
	}
	return result
}

// disconnectService tries to stop yggdrasil.service, when it hasn't
// been already stopped.
func disconnectService(disconnectResult *DisconnectResult, errorMessages *map[string]logging.LogMessage, uiSettings interactive.UserInterfaceSettings) error {
	// First check if the service hasn't been already stopped
	isInactive, err := activation.IsServiceInState("inactive")
	if err != nil {
		return err
	}
	if isInactive {
		infoMsg := fmt.Sprintf("The %s service is already inactive", config.ServiceName)
		disconnectResult.YggdrasilStopped = true
		interactive.InteractivePrintf(" [%v] %v\n", uiSettings, uiSettings.IconInfo, infoMsg)
		return nil
	}
	// When the service is not inactive, then try to get this service to this state
	progressMessage := fmt.Sprintf(" Deactivating the %v service", config.ServiceName)
	err = interactive.ShowProgress(progressMessage, activation.DeactivateService, interactive.SmallIndent, uiSettings)
	if err != nil {
		errMsg := fmt.Sprintf("Cannot deactivate %s service: %v", config.ServiceName, err)
		(*errorMessages)[config.ServiceName] = logging.LogMessage{
			Level:   slog.LevelError,
			Message: fmt.Errorf("%v", errMsg)}
		disconnectResult.YggdrasilStopped = false
		disconnectResult.YggdrasilStoppedError = errMsg
		return fmt.Errorf("%v", errMsg)
	} else {
		disconnectResult.YggdrasilStopped = true
		interactive.InteractivePrintf(" [%v] Deactivated the %v service\n", uiSettings, uiSettings.IconOK, config.ServiceName)
		return nil
	}
}

// disconnectFromInsights tries to unregister system from Red Hat Insights
func disconnectFromInsights(disconnectResult *DisconnectResult, errorMessages *map[string]logging.LogMessage, uiSettings interactive.UserInterfaceSettings) error {
	// 1. Check whether system is already disconnected from Insights
	isRegistered, err := insights.IsRegistered()
	if err != nil {
		disconnectResult.InsightsDisconnectedError = err.Error()
		(*errorMessages)["insights"] = logging.LogMessage{
			Level:   slog.LevelError,
			Message: err}
		return err
	}
	// When system is not registered to insights, then there is nothing to disconnect
	if !isRegistered {
		infoMsg := "This system is already disconnected from Red Hat Insights"
		disconnectResult.InsightsDisconnected = true
		interactive.InteractivePrintf(" [%v] %v\n", uiSettings, uiSettings.IconInfo, infoMsg)
		return nil
	}

	// 2. When system is registered to insights, then try to disconnect from insights
	progressMessage := " Disconnecting from Red Hat Insights"
	err = interactive.ShowProgress(progressMessage, insights.Unregister, interactive.SmallIndent, uiSettings)
	if err != nil {
		errMsg := fmt.Sprintf("Unable to disconnect from Red Hat Insights: %v", err)
		(*errorMessages)["insights"] = logging.LogMessage{
			Level:   slog.LevelError,
			Message: fmt.Errorf("%v", errMsg)}
		disconnectResult.InsightsDisconnected = false
		disconnectResult.InsightsDisconnectedError = errMsg
		return fmt.Errorf("%v", errMsg)
	} else {
		disconnectResult.InsightsDisconnected = true
		interactive.InteractivePrintf(" [%v] Disconnected from Red Hat Insights\n", uiSettings, uiSettings.IconOK)
		return nil
	}
}

// disconnectFromRHSM tries to unregister system from Red Hat Subscription Management
func disconnectFromRHSM(disconnectResult *DisconnectResult, errorMessages *map[string]logging.LogMessage, uiSettings interactive.UserInterfaceSettings) error {
	// 1. Check whether system is registered or not
	isRegistered, err := rhsm.IsRegistered()
	if err != nil {
		disconnectResult.RHSMDisconnectedError = err.Error()
		(*errorMessages)["rhsm"] = logging.LogMessage{
			Level:   slog.LevelError,
			Message: err}
		return err
	}
	// When system is not registered to RHSM, then there is nothing to disconnect
	if !isRegistered {
		infoMsg := "This system is already disconnected from Red Hat Subscription Management"
		disconnectResult.RHSMDisconnected = true
		interactive.InteractivePrintf(" [%v] %v\n", uiSettings, uiSettings.IconInfo, infoMsg)
		return nil
	}

	// 2. When system is registered to RHSM, then try to disconnect from RHSM
	progressMessage := " Disconnecting from Red Hat Subscription Management"
	err = interactive.ShowProgress(progressMessage, rhsm.Unregister, interactive.SmallIndent, uiSettings)
	if err != nil {
		errMsg := fmt.Sprintf("Unable to disconnect from Red Hat Subscription Management: %v", err)
		(*errorMessages)["rhsm"] = logging.LogMessage{
			Level:   slog.LevelError,
			Message: fmt.Errorf("%v", errMsg)}
		disconnectResult.RHSMDisconnected = false
		disconnectResult.RHSMDisconnectedError = errMsg
		return fmt.Errorf("%v", errMsg)
	} else {
		disconnectResult.RHSMDisconnected = true
		interactive.InteractivePrintf(" [%v] Disconnected from Red Hat Subscription Management\n", uiSettings, uiSettings.IconOK)
		return nil
	}
}

// disconnectAction tries to disconnect system from Red Hat Insights and Red Hat Subscription Management
func disconnectAction(ctx *cli.Context) error {
	uiSettings := interactive.ConfigureUISettings(ctx)

	var disconnectResult DisconnectResult
	durations := make(map[string]time.Duration)
	errorMessages := make(map[string]logging.LogMessage)

	disconnectResult.format = ctx.String("format")

	// Collect some basic information about the system.
	start := time.Now()
	hostname, err := os.Hostname()
	stop := time.Now()
	durations["hostname"] = stop.Sub(start)
	if err != nil {
		if uiSettings.IsMachineReadable {
			disconnectResult.HostnameError = err.Error()
		} else {
			slog.Error("unable to get hostname", "err", err)
		}
	} else {
		disconnectResult.Hostname = hostname
	}

	start = time.Now()
	uid := os.Getuid()
	stop = time.Now()
	durations["uid"] = stop.Sub(start)
	disconnectResult.UID = uid

	// When user is not root, then print only warning and continue
	if uid != 0 {
		warningMsg := fmt.Sprintf("not running as root user (UID %v), functionality may be limited", uid)
		if uiSettings.IsMachineReadable {
			disconnectResult.UIDError = warningMsg
		} else {
			fmt.Printf("%v %v\n", uiSettings.IconInfo, warningMsg)
		}
	}

	// 1. Disconnect service
	start = time.Now()
	_ = disconnectService(&disconnectResult, &errorMessages, uiSettings)
	stop = time.Now()
	durations["service-disconnect"] = stop.Sub(start)

	// 2. Disconnect from insights
	start = time.Now()
	_ = disconnectFromInsights(&disconnectResult, &errorMessages, uiSettings)
	stop = time.Now()
	durations["insights-disconnect"] = stop.Sub(start)

	// 3. Disconnect from Red Hat Subscription Management
	start = time.Now()
	_ = disconnectFromRHSM(&disconnectResult, &errorMessages, uiSettings)
	stop = time.Now()
	durations["rhsm-disconnect"] = stop.Sub(start)

	// Print durations when log level is debug
	interactive.ShowTimeDuration(durations)

	// Print possible error messages
	err = interactive.ShowErrorMessages("disconnect", errorMessages, uiSettings)
	if err != nil {
		return err
	}

	if uiSettings.IsMachineReadable {
		fmt.Print(disconnectResult.String())
	}

	return nil
}