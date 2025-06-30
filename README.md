# **🎙️ Mac Microphone Control with Arduino**

This project provides a solution to control your macOS system microphone's mute/unmute state using an Arduino board equipped with Modulino Buttons and Pixels modules. It offers a physical interface for microphone control, with visual feedback on its status.

## **1\. Project Overview**

The system comprises two main components that communicate via serial:

* **Go Application:** Runs on your Mac, detects the specific Arduino board, controls the system microphone, and synchronizes its state with the Arduino. It features automatic reconnection and a safety unmute on disconnect.  
* **Arduino Sketch:** Runs on the Arduino board, reads input from Modulino Buttons, controls the Modulino Pixels LED to reflect the microphone's status, and communicates with the Go application.

## **2\. Features**

### **Go Application**

* **macOS Microphone Control:** Mutes and unmutes the system microphone using osascript.  
* **Specific Arduino Detection:** Uses arduino-cli to automatically find and connect to a particular Arduino board based on its Vendor ID (VID) and Product ID (PID).  
* **Board Identification Protocol:** Implements a handshake protocol to ensure the connected Arduino is running the correct sketch.  
* **Automatic Reconnection:** Robustly handles disconnections of the Arduino board, automatically attempting to re-establish the connection.  
* **Auto-Unmute on Disconnect:** As a safety feature, the system microphone is automatically unmuted if the Arduino board disconnects.  
* **Microphone State Synchronization:** Sends the actual macOS microphone state to the Arduino to keep the Modulino Pixels LED synchronized.

### **Arduino Sketch**

* **Button Input:** Reads all 3 buttons on the Modulino Buttons module using the Button2 library for reliable, debounced input.  
* **Microphone Toggle Request:** Any button press on the Modulino Buttons module sends a request to the Go application to toggle the microphone state.  
* **Status LED:** The Modulino Pixels LED indicates the microphone's state:  
  * **RED** when the microphone is **UNMUTED**.  
  * **OFF** (black) when the microphone is **MUTED**.  
* **Serial Communication:** Exchanges commands and status updates with the Go application.

## **3\. Getting Started**

### **3.1 Dependencies**

**For the Go Application:**

* **Go:** Go runtime and compiler (version 1.16 or higher recommended).  
* **go.bug.st/serial:** Go library for serial communication.  
* **arduino-cli:** Command-line tool for interacting with Arduino boards. Must be installed and configured in your system's PATH.

**For the Arduino Sketch:**

* **Arduino IDE:** Or any compatible environment for uploading sketches.  
* **Arduino Board:** The target board for this project.  
* **Modulino Library:** Available via Arduino IDE's Library Manager or [GitHub](https://github.com/arduino-libraries/Modulino).  
* **Button2 Library:** Available via Arduino IDE's Library Manager or [GitHub](https://www.google.com/search?q=https://github.com/madhephaestus/Button2).

### **3.2 Hardware Setup**

* **Arduino Board:** Connect your Arduino board to your Mac via USB.  
* **Modulino Buttons:** Attach the Modulino Buttons module to your Arduino. Modulino modules typically connect via I2C, so ensure proper I2C pin connections if not using a dedicated Modulino carrier board.  
* **Modulino Pixels:** Attach the Modulino Pixels module to your Arduino. Similarly, ensure proper I2C connections.

### **3.3 Arduino Sketch Setup and Upload**

1. **Install Libraries:**  
   * Open the Arduino IDE.  
   * Go to Sketch \> Include Library \> Manage Libraries...  
   * Search for and install "Modulino".  
   * Search for and install "Button2".  
2. **Open Sketch:** Copy the Arduino sketch code (provided in the conversation history) into a new sketch in the Arduino IDE.
5. **Upload:** Click the "Upload" button to compile and upload the sketch to your Arduino board.

### **3.4 Go Application Setup and Configuration**

1. **Install Go:** If you don't have it, download and install Go for macOS from the official website: [https://go.dev/doc/install](https://go.dev/doc/install).  
2. **Install go.bug.st/serial:**  
   go get go.bug.st/serial

3. **Install arduino-cli:** Follow the official instructions to install arduino-cli on your Mac: [https://arduino.github.io/arduino-cli/latest/installation/](https://arduino.github.io/arduino-cli/latest/installation/). Ensure it's accessible from your system's PATH.  
4. **Find Your Arduino's VID and PID:**  
   * Connect your Arduino Board to your Mac.  
   * Open Terminal and run:  
     arduino-cli board list \--format json

   * Examine the JSON output for your board. For an Arduino UNO R4 WiFi, it will look something like this (focus on the port.properties section):
   ```
     {  
       "detected\_ports": \[  
         // ... other ports ...  
         {  
           "matching\_boards": \[  
             {  
               "name": "Arduino UNO R4 WiFi",  
               "fqbn": "arduino:renesas\_uno:unor4wifi"  
             }  
           \],  
           "port": {  
             "address": "/dev/cu.usbmodemDC5475D0E8BC2",  
             "label": "/dev/cu.usbmodemDC5475D0E8BC2",  
             "protocol": "serial",  
             "protocol\_label": "Serial Port (USB)",  
             "properties": {  
               "pid": "0x1002",  
               "serialNumber": "DC5475D0E8BC",  
               "vid": "0x2341"  
             },  
             "hardware\_id": "DC5475D0E8BC"  
           }  
         },  
         // ... other ports ...  
       \]  
     }
     ```

   * Note down the vid and pid values from the "properties" field (e.g., "0x2341" and "0x1002").  
5. **Update Go Code:** Open the Go application file (e.g., main.go or mac-microphone-control.go) and modify the targetArduinoVID and targetArduinoPID constants with the values you found:  
    ```
   const (  
       // ...  
       targetArduinoVID \= "0x2341" // \<--- Update with your Arduino's VID  
       targetArduinoPID \= "0x1002" // \<--- Update with your Arduino's PID  
       // ...  
   )
   ```

6. **Save and Run:** Save the Go file. In your terminal, navigate to the MuteMicrophone directory and run:
    ```
    go run main.go
    ```

   To compile a standalone executable:  
   ```
   go build
   ./MuteMicrophone
   ```

## **4\. Usage**

1. **Start Arduino:** Ensure your Arduino board with the sketch uploaded and Modulino modules connected is powered on.  
2. **Start Go Application:** Run the Go application on your Mac as described in Section 3.4.  
3. **Control Microphone:** Press any button on the Modulino Buttons module. This will:  
   * Send a toggle request to the Go application.  
   * The Go application will change the system microphone's mute state.  
   * The Go application will send the new microphone state back to the Arduino.  
   * The Modulino Pixels LED will update:  
     * **RED:** Microphone is UNMUTED.  
     * **OFF:** Microphone is MUTED.

## **5\. Communication Protocol**

The communication between the Go application and the Arduino sketch occurs over serial, using simple string commands terminated by a newline character (\\n).

### **5.1 Commands **

* MUTE: Mute the microphone
* UNMUTE: Unmute the microphone
* GET\_STATE: Request the current microphone state  
* IDENTIFY\_ARDUINO: Sent by the Go app during connection establishment to identify a specific Arduino running the correct sketch.

### **5.2 Responses (Arduino \-\> Go App)**

* MUTED: Reports that the microphone is muted.  
* UNMUTED: Reports that the microphone is unmuted.  
* ERROR: Indicates an error occurred during communication.  
* UNKNOWN\_COMMAND: Indicates an unrecognized command.  
* IDENTIFY\_ACK: Acknowledges the IDENTIFY\_ARDUINO command, confirming the board's identity.

## **6\. Troubleshooting**

* **"Error executing 'arduino-cli board list'":**  
  * Ensure arduino-cli is installed and its directory is added to your system's PATH.  
  * Try running arduino-cli board list directly in the terminal to see if it works.  
* **"No Arduino board found with VID:X and PID:Y":**  
  * Double-check that your Arduino is connected and powered on.  
  * Verify that the targetArduinoVID and targetArduinoPID constants in your Go code exactly match the output of arduino-cli board list \--format json.  
  * Ensure the Arduino IDE's serial monitor is closed, as it might hog the port.  
* **"Error opening serial port":**  
  * The port might be in use by another application (e.g., Arduino IDE Serial Monitor). Close any other applications using the serial port.  
  * You might not have sufficient permissions to access /dev/cu.\* ports.  
* **Buttons not responding / LED not updating:**  
  * Verify the Arduino sketch is uploaded correctly.  
  * Ensure the Go application is running and successfully connected to the Arduino (check its console output).  
  * Check your Modulino Buttons and Pixels connections to the Arduino, especially I2C (SDA/SCL) and power.
* **Debug Messages:** To see detailed debug messages from the Arduino, uncomment \#define DEBUG at the top of the Arduino sketch before uploading.

## **7\. Future Improvements**
* **Multiplatform support:** Support for other OS
* **UI for Go App:** Create a simple graphical user interface (GUI) for the Go application instead of just console output, providing a more user-friendly experience.  
* **Configuration File:** Allow VID/PID and other settings to be configured via a file instead of hardcoding in the Go application.