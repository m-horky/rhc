package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/urfave/cli/v2"
	"os"
	"time"

	"github.com/briandowns/spinner"
	systemd "github.com/coreos/go-systemd/v22/dbus"

	"github.com/redhatinsights/rhc/pkg/config"
	"github.com/redhatinsights/rhc/pkg/insights"
	"github.com/redhatinsights/rhc/pkg/interactive"
	"github.com/redhatinsights/rhc/pkg/rhsm"
)

// SystemStatus represents the status of the system's connection
type SystemStatus struct {
	RHSMConnected    bool `json:"rhsm_connected"`
	InsightsConnected bool `json:"insights_connected"`
	YggdrasilRunning bool `json:"yggdrasil_running"`
	returnCode       int
	format           string
}

// String returns string representation of SystemStatus
func (systemStatus SystemStatus) String() string {
	var result string
	switch systemStatus.format {
	case "json":
		data, err := json.MarshalIndent(systemStatus, "", "    ")
		if err != nil {
			return err.Error()
		}
		result = string(data)
	case "":
		break
	default:
		result = "error: unsupported document format: " + systemStatus.format
	}
	return result
}

// rhsmStatus tries to print status provided by RHSM D-Bus API. If we provide
// output in machine-readable format, then we only set files in SystemStatus
// structure and content of this structure will be printed later
func rhsmStatus(systemStatus *SystemStatus, uiSettings interactive.UserInterfaceSettings) error {

	uuid, err := rhsm.GetConsumerUUID()
	if err != nil {
		return fmt.Errorf("unable to get consumer UUID: %s", err)
	}
	if uuid == "" {
		systemStatus.returnCode += 1
		if uiSettings.IsMachineReadable {
			systemStatus.RHSMConnected = false
		} else {
			fmt.Printf("%v Not connected to Red Hat Subscription Management\n", uiSettings.IconInfo)
		}
	} else {
		if uiSettings.IsMachineReadable {
			systemStatus.RHSMConnected = true
		} else {
			fmt.Printf("%v Connected to Red Hat Subscription Management\n", uiSettings.IconOK)
		}
	}
	return nil
}

// insightsStatus tries to print status provided by insights-client --status command.
func insightsStatus(systemStatus *SystemStatus, uiSettings interactive.UserInterfaceSettings) error {
	isRegistered, err := insights.IsRegistered()
	if err != nil {
		if uiSettings.IsMachineReadable {
			systemStatus.InsightsConnected = false
		} else {
			fmt.Printf("%v Unable to get status of connection to Red Hat Insights: %v\n", uiSettings.IconError, err)
		}
		systemStatus.returnCode += 1
		return err
	}
	if !isRegistered {
		if uiSettings.IsMachineReadable {
			systemStatus.InsightsConnected = false
		} else {
			fmt.Printf("%v Not connected to Red Hat Insights\n", uiSettings.IconInfo)
		}
		systemStatus.returnCode += 1
	} else {
		if uiSettings.IsMachineReadable {
			systemStatus.InsightsConnected = true
		} else {
			fmt.Printf("%v Connected to Red Hat Insights\n", uiSettings.IconOK)
		}
	}
	return nil
}

// yggdrasilStatus tries to print status of yggdrasil.service using systemd D-Bus API
func yggdrasilStatus(systemStatus *SystemStatus, uiSettings interactive.UserInterfaceSettings) error {
	conn, err := systemd.NewSystemdConnectionContext(context.Background())
	if err != nil {
		return fmt.Errorf("unable to connect to systemd: %v", err)
	}
	defer conn.Close()

	unitStatuses, err := conn.ListUnitsContext(context.Background())
	if err != nil {
		return fmt.Errorf("unable to list systemd units: %v", err)
	}
	var serviceFound bool = false
	for _, unitStatus := range unitStatuses {
		if unitStatus.Name == config.ServiceName+".service" {
			serviceFound = true
			if uiSettings.IsMachineReadable {
				if unitStatus.ActiveState == "active" {
					systemStatus.YggdrasilRunning = true
				} else {
					systemStatus.YggdrasilRunning = false
				}
			} else {
				switch unitStatus.ActiveState {
				case "active":
					fmt.Printf("%v %v service is running\n", uiSettings.IconOK, config.ServiceName)
				case "inactive":
					fmt.Printf("%v %v service is not running\n", uiSettings.IconInfo, config.ServiceName)
					systemStatus.returnCode += 1
				case "failed":
					fmt.Printf("%v %v service has failed to start\n", uiSettings.IconError, config.ServiceName)
					systemStatus.returnCode += 1
				default:
					fmt.Printf("%v %v service is in an unknown state (%s)\n", uiSettings.IconError, config.ServiceName, unitStatus.ActiveState)
					systemStatus.returnCode += 1
				}
			}
			break
		}
	}
	if !serviceFound {
		if uiSettings.IsMachineReadable {
			systemStatus.YggdrasilRunning = false
		} else {
			fmt.Printf("%v Unable to find %v service\n", uiSettings.IconError, config.ServiceName)
		}
		systemStatus.returnCode += 1
	}
	return nil
}

// statusAction tries to print status of connection to the Red Hat Subscription Management,
// Red Hat Insights and status of yggdrasil service
func statusAction(ctx *cli.Context) error {
	uiSettings := interactive.ConfigureUISettings(ctx)

	var systemStatus SystemStatus

	systemStatus.format = ctx.String("format")

	var s *spinner.Spinner
	if uiSettings.IsRich {
		s = spinner.New(spinner.CharSets[9], 100*time.Millisecond)
		s.Prefix = interactive.SmallIndent + "["
		s.Suffix = "] Checking status..."
		s.Start()
		defer s.Stop()
	}

	err := rhsmStatus(&systemStatus, uiSettings)
	if err != nil {
		fmt.Printf("%v %v\n", uiSettings.IconError, err.Error())
	}

	err = insightsStatus(&systemStatus, uiSettings)
	if err != nil {
		fmt.Printf("%v %v\n", uiSettings.IconError, err.Error())
	}

	err = yggdrasilStatus(&systemStatus, uiSettings)
	if err != nil {
		fmt.Printf("%v %v\n", uiSettings.IconError, err.Error())
	}

	if uiSettings.IsRich {
		s.Stop()
	}

	if uiSettings.IsMachineReadable {
		fmt.Print(systemStatus.String())
	}

	os.Exit(systemStatus.returnCode)

	return nil
}