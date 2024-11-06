package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/briandowns/spinner"
	systemd "github.com/coreos/go-systemd/v22/dbus"
	"github.com/subpop/go-log"
)

var StatePath = "/var/lib/rhc/state.json"

// FIXME The State struct does not cover partial connection (e.g. failed during the process)

type State struct {
	Connected bool `json:"connected"`
}

// GetState loads a program state from a cached file.
func GetState() *State {
	content, err := os.ReadFile(StatePath)
	if err != nil {
		log.Errorf("Could not read state cache: %v", err)
		var state = &State{Connected: false}
		if err = state.Save(); err != nil {
			panic(err)
		}
		return state
	}

	// set up defaults
	var state = State{Connected: false}

	// read in actual state
	if err = json.Unmarshal(content, &state); err != nil {
		panic(err)
	}
	return &state
}

// Save writes the program state into a cache file.
func (state *State) Save() error {
	content, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return os.WriteFile(StatePath, content, 0644)
}

// rhsmStatus tries to print status provided by RHSM D-Bus API. If we provide
// output in machine-readable format, then we only set files in SystemStatus
// structure and content of this structure will be printed later
func rhsmStatus(systemStatus *SystemStatus) error {

	uuid, err := getConsumerUUID()
	if err != nil {
		return fmt.Errorf("unable to get consumer UUID: %s", err)
	}
	if uuid == "" {
		systemStatus.returnCode += 1
		if uiSettings.isMachineReadable {
			systemStatus.RHSMConnected = false
		} else {
			fmt.Printf("%v Not connected to Red Hat Subscription Management\n", uiSettings.iconInfo)
		}
	} else {
		if uiSettings.isMachineReadable {
			systemStatus.RHSMConnected = true
		} else {
			fmt.Printf("%v Connected to Red Hat Subscription Management\n", uiSettings.iconOK)
		}
	}
	return nil
}

// insightStatus tries to print status of insights client
func insightStatus(systemStatus *SystemStatus) {
	var s *spinner.Spinner
	if uiSettings.isRich {
		s = spinner.New(spinner.CharSets[9], 100*time.Millisecond)
		s.Suffix = " Checking Red Hat Insights..."
		s.Start()
	}
	isRegistered, err := insightsIsRegistered()
	if uiSettings.isRich {
		s.Stop()
	}
	if isRegistered {
		if uiSettings.isMachineReadable {
			systemStatus.InsightsConnected = true
		} else {
			fmt.Print(uiSettings.iconOK + " Connected to Red Hat Insights\n")
		}
	} else {
		systemStatus.returnCode += 1
		if err == nil {
			if uiSettings.isMachineReadable {
				systemStatus.InsightsConnected = false
			} else {
				fmt.Print(uiSettings.iconInfo + " Not connected to Red Hat Insights\n")
			}
		} else {
			if uiSettings.isMachineReadable {
				systemStatus.InsightsConnected = false
				systemStatus.InsightsError = err.Error()
			} else {
				fmt.Printf(uiSettings.iconError+" Cannot detect Red Hat Insights status: %v\n", err)
			}
		}
	}
}

// serviceStatus tries to print status of yggdrasil.service or rhcd.service
func serviceStatus(systemStatus *SystemStatus) error {
	ctx := context.Background()
	conn, err := systemd.NewSystemConnectionContext(ctx)
	if err != nil {
		systemStatus.YggdrasilRunning = false
		systemStatus.YggdrasilError = err.Error()
		return fmt.Errorf("unable to connect to systemd: %s", err)
	}
	defer conn.Close()
	unitName := ServiceName + ".service"
	properties, err := conn.GetUnitPropertiesContext(ctx, unitName)
	if err != nil {
		systemStatus.YggdrasilRunning = false
		systemStatus.YggdrasilError = err.Error()
		return fmt.Errorf("unable to get properties of %s: %s", unitName, err)
	}
	activeState := properties["ActiveState"]
	if activeState.(string) == "active" {
		if uiSettings.isMachineReadable {
			systemStatus.YggdrasilRunning = true
		} else {
			fmt.Printf(uiSettings.iconOK+" The %v service is active\n", ServiceName)
		}
	} else {
		systemStatus.returnCode += 1
		if uiSettings.isMachineReadable {
			systemStatus.YggdrasilRunning = false
		} else {
			fmt.Printf(uiSettings.iconInfo+" The %v service is inactive\n", ServiceName)
		}
	}
	return nil
}
