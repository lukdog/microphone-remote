package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"

	"go.bug.st/serial" // Imports the serial communication library
)

// Defines serial communication commands and responses
const (
	cmdMute         = "MUTE"
	cmdUnmute       = "UNMUTE"
	cmdGetState     = "GET_STATE"
	cmdIdentify     = "IDENTIFY_ARDUINO"
	respMuted       = "MUTED"
	respUnmuted     = "UNMUTED"
	respError       = "ERROR"
	respUnknown     = "UNKNOWN_COMMAND"
	respIdentifyAck = "IDENTIFY_ACK"

	serialBaud = 9600

	// Identifiers for your specific Arduino.
	// You can find these values by running 'arduino-cli board list --format json'
	// in your terminal when your Arduino is connected.
	// Example for Arduino Uno R4 WiFi from your output: VID = "0x2341", PID = "0x1002"
	// Ensure these match the output you see.
	targetArduinoVID = "0x2341" // <--- CHANGE THIS TO YOUR ARDUINO'S VID
	targetArduinoPID = "0x1002" // <--- CHANGE THIS TO YOUR ARDUINO'S PID

	identificationTimeout = 3 * time.Second // Timeout for identification response
)

// PortProperties contains detailed properties of the serial port, including VID and PID.
type PortProperties struct {
	PID          string `json:"pid"`
	SerialNumber string `json:"serialNumber"`
	VID          string `json:"vid"`
}

// Port represents the serial port information.
type Port struct {
	Address       string         `json:"address"`
	Label         string         `json:"label"`
	Protocol      string         `json:"protocol"`
	ProtocolLabel string         `json:"protocol_label"`
	Properties    PortProperties `json:"properties"`
	HardwareID    string         `json:"hardware_id"`
}

// MatchingBoard represents information about a board matching the port.
type MatchingBoard struct {
	Name string `json:"name"`
	Fqbn string `json:"fqbn"`
}

// DetectedPortItem represents an item within the "detected_ports" array.
// It might contain matching_boards or just port info.
type DetectedPortItem struct {
	MatchingBoards []MatchingBoard `json:"matching_boards"` // Optional, only for some ports (can be empty)
	Port           Port            `json:"port"`
}

// ArduinoCLIResponse is the top-level structure for the entire JSON output from arduino-cli.
type ArduinoCLIResponse struct {
	DetectedPorts []DetectedPortItem `json:"detected_ports"`
}

// setMicrophoneMuteState sets the mute state of the system microphone on macOS.
// It uses osascript to interact with audio settings.
func setMicrophoneMuteState(mute bool) error {
	var script string
	if mute {
		script = `set volume input volume 0` // Mute the microphone
	} else {
		script = `set volume input volume 100` // Unmute the microphone (sets to 100%, can be adjusted)
	}

	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error executing osascript to set state: %v\nOutput: %s", err, output)
	}
	log.Printf("Microphone set to Mute: %t. osascript output: %s", mute, strings.TrimSpace(string(output)))
	return nil
}

// getMicrophoneMuteState retrieves the mute state of the system microphone on macOS.
// It uses osascript to query the audio state.
func getMicrophoneMuteState() (bool, error) {
	// AppleScript to get input volume.
	// If input volume is 0, the microphone is considered muted.
	script := `get volume settings`

	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("error executing osascript to get state: %v\nOutput: %s", err, output)
	}

	// The output will be something like "output volume:40, input volume:100, alert volume:100, output muted:false, input muted:false"
	// We need to parse the string to find "input volume:" and its value.
	outputStr := strings.TrimSpace(string(output))
	log.Printf("osascript output for get state: %s", outputStr)

	// Search for the string "input volume:"
	inputVolumeIndex := strings.Index(outputStr, "input volume:")
	if inputVolumeIndex == -1 {
		return false, fmt.Errorf("cannot find 'input volume' in osascript output")
	}

	// Extract the substring after "input volume:"
	sub := outputStr[inputVolumeIndex+len("input volume:"):]
	// Find the end of the input volume number
	endIndex := strings.IndexAny(sub, ", \n") // Search for comma, space or newline
	if endIndex == -1 {
		endIndex = len(sub) // If no delimiters, the rest is the number
	}
	volumeStr := strings.TrimSpace(sub[:endIndex])

	var volume int
	_, err = fmt.Sscanf(volumeStr, "%d", &volume)
	if err != nil {
		return false, fmt.Errorf("cannot parse input volume '%s': %v", volumeStr, err)
	}

	// If input volume is 0, we consider the microphone muted.
	isMuted := (volume == 0)
	return isMuted, nil
}

// findSpecificArduinoPort executes arduino-cli to find the port of a specific Arduino board
func findSpecificArduinoPort(targetVID, targetPID string) (string, error) {
	log.Println("Searching for specific Arduino port using arduino-cli...")
	cmd := exec.Command("arduino-cli", "board", "list", "--format", "json")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("error executing 'arduino-cli board list': %v\nOutput: %s\nPlease ensure arduino-cli is installed and configured", err, output)
	}

	// Unmarshal the JSON output into the top-level ArduinoCLIResponse struct
	var cliResponse ArduinoCLIResponse
	err = json.Unmarshal(output, &cliResponse)
	if err != nil {
		return "", fmt.Errorf("error parsing arduino-cli JSON output: %v\nOutput: %s", err, output)
	}

	log.Printf("Detected %d ports by arduino-cli...", len(cliResponse.DetectedPorts))

	for _, item := range cliResponse.DetectedPorts {
		// Check if the current item has port properties (not all port types will, e.g., debug-console)
		// and if it matches the target VID/PID
		if item.Port.Properties.VID == targetVID && item.Port.Properties.PID == targetPID {
			// You can optionally log the matching board name from MatchingBoards if available
			boardName := "Unknown Board"
			if len(item.MatchingBoards) > 0 {
				boardName = item.MatchingBoards[0].Name
			}
			log.Printf("Found potential Arduino board: %s (VID:%s, PID:%s) on port: %s", boardName, item.Port.Properties.VID, item.Port.Properties.PID, item.Port.Address)
			return item.Port.Address, nil
		}
	}

	return "", fmt.Errorf("no Arduino board found with VID: %s and PID: %s. Please ensure it is connected and 'arduino-cli' can detect it", targetVID, targetPID)
}

// identifyArduino sends CMD_IDENTIFY to the Arduino and waits for RESP_IDENTIFY_ACK within a timeout.
// This confirms the connected board is running the expected sketch.
func identifyArduino(port serial.Port) error {
	log.Println("Attempting to identify Arduino by sending IDENTIFY_ARDUINO command...")

	// Create a new reader for this specific identification attempt
	reader := bufio.NewReader(port)

	// Send the IDENTIFY_ARDUINO command
	_, err := port.Write([]byte(cmdIdentify + "\n"))
	if err != nil {
		return fmt.Errorf("error sending IDENTIFY_ARDUINO for identification: %v", err)
	}

	responseChan := make(chan string, 1)
	errChan := make(chan error, 1)

	// Goroutine to read the response to prevent blocking the main loop
	go func() {
		line, readErr := reader.ReadString('\n')
		if readErr != nil {
			errChan <- fmt.Errorf("error reading response for identification: %v", readErr)
			return
		}
		responseChan <- strings.TrimSpace(strings.ToUpper(line))
	}()

	select {
	case response := <-responseChan:
		if response == respIdentifyAck {
			log.Println("Arduino successfully identified with IDENTIFY_ACK.")
			return nil
		}
		return fmt.Errorf("received unexpected response for identification: '%s'", response)
	case <-time.After(identificationTimeout):
		return fmt.Errorf("timeout waiting for Arduino identification (IDENTIFY_ACK) response after %s", identificationTimeout)
	}
}

func main() {
	log.Println("Go application for microphone control started.")

	// Outer loop to handle serial port reconnection
	for {
		log.Println("Attempting to connect to Arduino...")
		portName, err := findSpecificArduinoPort(targetArduinoVID, targetArduinoPID)
		if err != nil {
			log.Printf("Unable to find Arduino port with matching VID/PID: %v. Retrying in 5 seconds...", err)
			time.Sleep(5 * time.Second) // Wait before retrying search
			continue                    // Go back to the beginning of the outer loop to retry
		}

		// Serial port configuration mode
		mode := &serial.Mode{
			BaudRate: serialBaud,
			Parity:   serial.NoParity,
			DataBits: 8,
			StopBits: serial.OneStopBit,
		}

		log.Printf("Attempting to open serial port: %s at %d baud...", portName, serialBaud)
		port, err := serial.Open(portName, mode)
		if err != nil {
			log.Printf("Error opening serial port %s: %v. Retrying in 5 seconds...", portName, err)
			time.Sleep(5 * time.Second) // Wait before retrying open
			continue                    // Go back to the beginning of the outer loop to retry
		}
		log.Println("Serial port opened successfully.")

		// Try to identify the Arduino by sending a specific command and waiting for acknowledgment.
		err = identifyArduino(port)
		if err != nil {
			log.Printf("Arduino identification failed: %v. Closing port and retrying...", err)
			port.Close()                // Close the port if identification fails
			time.Sleep(1 * time.Second) // Small delay before next retry
			continue                    // Go back to the outer loop to find another port
		}

		// If identification successful, get the actual microphone state from macOS
		// and send it to the Arduino to synchronize its LED.
		currentSystemMuteState, err := getMicrophoneMuteState()
		if err != nil {
			log.Printf("Error retrieving current system microphone state for initial sync: %v. Proceeding without initial sync.", err)
			// Continue even if initial sync fails, but log the error
		} else {
			var initialSyncResp string
			if currentSystemMuteState {
				initialSyncResp = respMuted
			} else {
				initialSyncResp = respUnmuted
			}
			_, writeErr := port.Write([]byte(initialSyncResp + "\n"))
			if writeErr != nil {
				log.Printf("Error sending initial system microphone state to Arduino via serial: %v. Closing port and retrying...", writeErr)
				port.Close()
				continue // Force the outer loop to retry
			}
			log.Printf("Initial system microphone state sent to Arduino: %s", initialSyncResp)
		}

		reader := bufio.NewReader(port) // Re-initialize reader for the main loop

		// This loop continues as long as the connection is stable
	inner:
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				log.Printf("Error reading from serial port (device likely disconnected): %v. Attempting to unmute microphone...", err)
				// Attempt to unmute the microphone
				unmuteErr := setMicrophoneMuteState(false)
				if unmuteErr != nil {
					log.Printf("Error unmuting microphone after board disconnection: %v", unmuteErr)
				} else {
					log.Println("Microphone unmuted due to board disconnection.")
				}
				port.Close() // Close the current (now erroneous) port
				break inner  // Exit the inner loop to re-enter the outer loop and attempt reconnection
			}

			command := strings.TrimSpace(strings.ToUpper(line))
			log.Printf("Command received from serial: '%s'", command)

			var response string
			var newState bool // true for muted, false for unmuted

			switch command {
			case cmdMute:
				err = setMicrophoneMuteState(true)
				if err != nil {
					response = respError
					log.Printf("Error muting microphone: %v", err)
				} else {
					response = respMuted
					log.Println("Microphone muted successfully.")
				}
				// Send the response only if writing does not immediately generate an error
				_, writeErr := port.Write([]byte(response + "\n"))
				if writeErr != nil {
					log.Printf("Error sending response via serial: %v", writeErr)
					// Attempt to unmute the microphone before closing the port
					unmuteErr := setMicrophoneMuteState(false)
					if unmuteErr != nil {
						log.Printf("Error unmuting microphone after serial write failure: %v", unmuteErr)
					} else {
						log.Println("Microphone unmuted due to serial write failure.")
					}
					port.Close() // Close and force reconnection
					break inner
				}

			case cmdUnmute:
				err = setMicrophoneMuteState(false)
				if err != nil {
					response = respError
					log.Printf("Error unmuting microphone: %v", err)
				} else {
					response = respUnmuted
					log.Println("Microphone unmuted successfully.")
				}
				_, writeErr := port.Write([]byte(response + "\n"))
				if writeErr != nil {
					log.Printf("Error sending response via serial: %v", writeErr)
					// Attempt to unmute the microphone before closing the port
					unmuteErr := setMicrophoneMuteState(false)
					if unmuteErr != nil {
						log.Printf("Error unmuting microphone after serial write failure: %v", unmuteErr)
					} else {
						log.Println("Microphone unmuted due to serial write failure.")
					}
					port.Close()
					break inner
				}

			case cmdGetState:
				// Arduino is asking for the current state.
				newState, err = getMicrophoneMuteState()
				if err != nil {
					response = respError
					log.Printf("Error retrieving microphone state for Arduino's GET_STATE request: %v", err)
				} else {
					if newState {
						response = respMuted
					} else {
						response = respUnmuted
					}
					log.Printf("Responding to Arduino's GET_STATE request with: %s (actual macOS state).", response)
				}
				_, writeErr := port.Write([]byte(response + "\n"))
				if writeErr != nil {
					log.Printf("Error sending response to Arduino's GET_STATE request via serial: %v", writeErr)
					// Attempt to unmute the microphone before closing the port
					unmuteErr := setMicrophoneMuteState(false)
					if unmuteErr != nil {
						log.Printf("Error unmuting microphone after serial write failure: %v", unmuteErr)
					} else {
						log.Println("Microphone unmuted due to serial write failure.")
					}
					port.Close()
					break inner
				}

			default:
				response = respUnknown
				log.Printf("Unknown command received: '%s'", command)
				_, writeErr := port.Write([]byte(response + "\n"))
				if writeErr != nil {
					log.Printf("Error sending response via serial: %v", writeErr)
					// Attempt to unmute the microphone before closing the port
					unmuteErr := setMicrophoneMuteState(false)
					if unmuteErr != nil {
						log.Printf("Error unmuting microphone after serial write failure: %v", unmuteErr)
					} else {
						log.Println("Microphone unmuted due to serial write failure.")
					}
					port.Close()
					break inner
				}
			}

			time.Sleep(50 * time.Millisecond)
		}

		// If the inner loop breaks, wait a moment before trying to reconnect
		time.Sleep(1 * time.Second)
	}
}
