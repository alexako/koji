/***************************************************
 * ESP32-Eyes for GC9A01 Round Display
 * Connects to Koji brain server for emotional state
 ***************************************************/

#include <WiFi.h>
#include <HTTPClient.h>
#include <ArduinoJson.h>
#include <TFT_eSPI.h>
#include "Face.h"
#include "FaceEmotions.hpp"

// Try to include local config first, fall back to default
#if __has_include("config_local.h")
#include "config_local.h"
#else
#include "config.h"
#endif

TFT_eSPI tft = TFT_eSPI();

#ifdef USE_DOUBLE_BUFFER
TFT_eSprite frameBuffer = TFT_eSprite(&tft);
#endif

Face *face;
unsigned long lastPoll = 0;
eEmotions currentEmotion = eEmotions::Normal;
bool wifiConnected = false;

void setupWiFi() {
  Serial.print("Connecting to WiFi");
  WiFi.mode(WIFI_STA);
  WiFi.begin(WIFI_SSID, WIFI_PASSWORD);
  
  int attempts = 0;
  while (WiFi.status() != WL_CONNECTED && attempts < 20) {
    delay(500);
    Serial.print(".");
    attempts++;
  }
  
  if (WiFi.status() == WL_CONNECTED) {
    wifiConnected = true;
    Serial.println();
    Serial.print("Connected! IP: ");
    Serial.println(WiFi.localIP());
  } else {
    Serial.println();
    Serial.println("WiFi connection failed - running in standalone mode");
  }
}

eEmotions indexToEmotion(int index) {
  if (index >= 0 && index < eEmotions::EMOTIONS_COUNT) {
    return static_cast<eEmotions>(index);
  }
  return eEmotions::Normal;
}

void pollBrainState() {
  if (!wifiConnected || WiFi.status() != WL_CONNECTED) {
    return;
  }

  HTTPClient http;
  http.begin(BRAIN_URL);
  http.setTimeout(1000);
  
  int httpCode = http.GET();
  
  if (httpCode == HTTP_CODE_OK) {
    String payload = http.getString();
    
    // Parse JSON response
    StaticJsonDocument<256> doc;
    DeserializationError error = deserializeJson(doc, payload);
    
    if (!error) {
      int emotionIndex = doc["emotion_index"] | 0;
      eEmotions newEmotion = indexToEmotion(emotionIndex);
      
      // Only update if emotion changed
      if (newEmotion != currentEmotion) {
        currentEmotion = newEmotion;
        face->Behavior.GoToEmotion(currentEmotion);
        
        Serial.print("Emotion changed to: ");
        Serial.print(emotionIndex);
        Serial.print(" (");
        Serial.print(doc["face_emotion"].as<const char*>());
        Serial.println(")");
      }
    } else {
      Serial.print("JSON parse error: ");
      Serial.println(error.c_str());
    }
  } else if (httpCode > 0) {
    Serial.print("HTTP error: ");
    Serial.println(httpCode);
  } else {
    Serial.print("Connection error: ");
    Serial.println(http.errorToString(httpCode));
  }
  
  http.end();
}

void setup(void) {
  Serial.begin(115200);
  delay(500);
  Serial.println("ESP32 Eyes - GC9A01");
  Serial.println("Koji Face Display v1.0");

  // Initialize display
  tft.init();
  tft.setRotation(0);
  tft.fillScreen(TFT_BLACK);

#ifdef USE_DOUBLE_BUFFER
  // Create sprite in PSRAM for double buffering
  frameBuffer.createSprite(SCREEN_WIDTH, SCREEN_HEIGHT);
  frameBuffer.fillSprite(TFT_BLACK);
  Serial.println("Double buffering enabled (PSRAM)");
#endif

  // Initialize face
  face = new Face(SCREEN_WIDTH, SCREEN_HEIGHT, EYE_SIZE);
  
  face->Expression.GoTo_Normal();
  face->RandomBehavior = false;  // Brain controls behavior
  face->RandomBlink = true;
  face->Blink.Timer.SetIntervalMillis(3000);
  face->RandomLook = true;
  face->Look.Timer.SetIntervalMillis(2000);

  // Connect to WiFi
  setupWiFi();
  
  Serial.println("Setup complete");
}

void loop() {
  // Poll brain server periodically
  unsigned long now = millis();
  if (now - lastPoll >= POLL_INTERVAL_MS) {
    lastPoll = now;
    pollBrainState();
  }

  // Update display
#ifdef USE_DOUBLE_BUFFER
  frameBuffer.fillSprite(TFT_BLACK);
  face->Update();
  frameBuffer.pushSprite(0, 0);
#else
  tft.fillScreen(TFT_BLACK);
  face->Update();
#endif
  delay(30);
}
