// SPDX-FileCopyrightText: 2022 Tillitis AB <tillitis.se>
// SPDX-License-Identifier: BSD-2-Clause

package tkeyclient

import (
	"fmt"
	"os"

	"go.bug.st/serial/enumerator"
)

const (
	tillitisMTAUSBV1VID  = "1207"
	tillitisMTAUSBV1PID  = "8887"
	tillitisTKEYUSBV2VID = "1209"
	tillitisTKEYUSBV2PID = "8885"
	// Custom errors
	ErrNoDevice    = constError("no TKey connected")
	ErrManyDevices = constError("more than one TKey connected")
)

type SerialPort struct {
	DevPath      string
	SerialNumber string
}

func isTKey(vid string, pid string) bool {
	return (vid == tillitisMTAUSBV1VID && pid == tillitisMTAUSBV1PID) ||
		(vid == tillitisTKEYUSBV2VID && pid == tillitisTKEYUSBV2PID)
}

// DetectSerialPort tries to detect an inserted TKey and returns the
// device path if successful.
func DetectSerialPort(verbose bool) (string, error) {
	ports, err := GetSerialPorts()
	if err != nil {
		return "", err
	}
	if len(ports) == 0 {
		if verbose {
			fmt.Fprintf(os.Stderr, "No TKey serial ports detected.\n")
		}
		return "", ErrNoDevice
	}
	if len(ports) > 1 {
		if verbose {
			fmt.Fprintf(os.Stderr, "Detected %d TKey serial ports:\n", len(ports))
			for _, p := range ports {
				fmt.Fprintf(os.Stderr, "%s with serial number %s\n", p.DevPath, p.SerialNumber)
			}
		}
		return "", ErrManyDevices
	}
	if verbose {
		fmt.Fprintf(os.Stderr, "Auto-detected serial port %s\n", ports[0].DevPath)
	}

	return ports[0].DevPath, nil
}

// GetSerialPorts enumerates any existing TKey serial ports identified
// on the system.
func GetSerialPorts() ([]SerialPort, error) {
	var ports []SerialPort

	portDetails, err := enumerator.GetDetailedPortsList()
	if err != nil {
		return nil, fmt.Errorf("GetDetailedPortsList: %w", err)
	}
	if len(portDetails) == 0 {
		return ports, nil
	}
	for _, port := range portDetails {
		if port.IsUSB && isTKey(port.VID, port.PID) {
			ports = append(ports, SerialPort{port.Name, port.SerialNumber})
		}
	}
	return ports, nil
}
