// Comments and log messages are in English.
// Serial commands and responses are in Italian to match the Go application.

// Uncomment the line below to enable debug messages on Serial Monitor.
//#define DEBUG

// Include the Modulino library
#include <Modulino.h>
// Include the Button2 library for robust button handling
#include <Button2.h>

using ButtonStateHandlerPtr = uint8_t (*)();

// Define the serial baud rate. Must match the Go application's baud rate.
const long BAUD_RATE = 9600;

// Serial commands and responses (must match Go app constants)
const String CMD_MUTE      = "MUTE";
const String CMD_UNMUTE    = "UNMUTE";
const String CMD_GET_STATE = "GET_STATE";
const String CMD_IDENTIFY  = "IDENTIFY_ARDUINO";
const String RESP_MUTED    = "MUTED";
const String RESP_UNMUTED  = "UNMUTED";
const String RESP_ERROR    = "ERROR";
const String RESP_UNKNOWN  = "UNKNOWN_COMMAND";
const String RESP_IDENTIFY = "IDENTIFY_ACK";

// Variable to store the current microphone state (true = muted, false = unmuted)
// This state should always reflect the actual system microphone state as reported by the Go app.
bool isMicrophoneMuted = false;

// Declare Modulino objects
ModulinoButtons buttons;
ModulinoPixels pixel;

// Array of Button2 objects, one for each button on the ModulinoButtons module.
Button2 button[3];

// Variable to store incoming serial data
String incomingSerialData = "";

uint8_t button0StateHandler() {
  buttons.update();
  return buttons.isPressed(0) ? LOW : HIGH;  // fake a normal button -> LOW = pressed
}

uint8_t button1StateHandler() {
  buttons.update();
  return buttons.isPressed(1) ? LOW : HIGH;  // fake a normal button -> LOW = pressed
}

uint8_t button2StateHandler() {
  buttons.update();
  return buttons.isPressed(2) ? LOW : HIGH;  // fake a normal button -> LOW = pressed
}

ButtonStateHandlerPtr buttonStateHandlers[] = {
  button0StateHandler,
  button1StateHandler,
  button2StateHandler
};

// Function to handle a button Click event
void handleButtonClick(Button2& btn) {
 
  if (isMicrophoneMuted) {
    Serial.println(CMD_UNMUTE);
    #ifdef DEBUG
    Serial.println("DEBUG: Sending UNMUTE command to Go app.");
    #endif
  } else {
    Serial.println(CMD_MUTE);
    #ifdef DEBUG
    Serial.println("DEBUG: Sending MUTE command to Go app.");
    #endif
  }
}

// Function to set the same color to all leds of Modulino Pixels
void setColorForAllLeds(uint8_t r, uint8_t g, uint8_t b){
  for(int i= 0; i<8; i++){
    pixel.set(i, r, g, b); 
  }
  pixel.show();
}

void setup() {
  // Initialize serial communication
  Serial.begin(BAUD_RATE);
  while(!Serial.available()){
    delay(100);
  }

  Modulino.begin();
 
  // Initialize Modulino Buttons
  buttons.begin();

  // Initialize Modulino Pixels
  pixel.begin();

  // Ensure the pixel is initially off (muted state)
  setColorForAllLeds(0,0,0);
  
  // Initialize Button2 objects for each button
  for (int i = 0; i < 3; i++) {
    button[i] = Button2();
    button[i].setButtonStateFunction(buttonStateHandlers[i]);
    button[i].setClickHandler(handleButtonClick); 
    button[i].setDebounceTime(35); 
    button[i].begin(BTN_VIRTUAL_PIN);
  }

  // Request the initial microphone state from the Go application.
  Serial.println(CMD_GET_STATE);
}

void loop() {

  // Call loop() for each Button2 object to process their states.
  for (int i = 0; i < 3; i++) {
    button[i].loop();
  }

  // Read any incoming serial data from the Go application, one character at a time.
  if (Serial.available()) {
    char inChar = (char)Serial.read();
    incomingSerialData += inChar; // Append character to the string

    // If a newline character is received, it means a full message has arrived
    if (inChar == '\n') {
      // Trim whitespace and convert to uppercase for robust comparison
      String receivedCommand = incomingSerialData;
      receivedCommand.trim();
      receivedCommand.toUpperCase();

      #ifdef DEBUG
      Serial.print("DEBUG: Received from Go app: '");
      Serial.print(receivedCommand);
      Serial.println("'");
      #endif

      // Check the received command and update microphone state and Modulino Pixels LED
      if (receivedCommand == RESP_MUTED) {
        isMicrophoneMuted = true;
        setColorForAllLeds(0, 0, 0);
        #ifdef DEBUG
        Serial.println("Microphone is now MUTED (Pixel OFF)."); // Status message, always visible
        #endif
      } else if (receivedCommand == RESP_UNMUTED) {
        isMicrophoneMuted = false;
        setColorForAllLeds(255, 0, 0);
        #ifdef DEBUG
        Serial.println("Microphone is now UNMUTED (Pixel RED)."); // Status message, always visible
        #endif
      } else if (receivedCommand == RESP_ERROR) {
        #ifdef DEBUG
        Serial.println("Go app reported an ERROR."); // Error message, always visible
        #endif
      } else if (receivedCommand == RESP_UNKNOWN) {
        #ifdef DEBUG
        Serial.println("Go app received an UNKNOWN_COMMAND from Arduino."); // Error message, always visible
        #endif
      } else if (receivedCommand == CMD_IDENTIFY) {
        Serial.println(RESP_IDENTIFY);
      }

      // Clear the string for the next incoming message
      incomingSerialData = "";
    }
  }

  // A small delay to keep the loop from running too fast, reducing CPU usage
  delay(10);
}
