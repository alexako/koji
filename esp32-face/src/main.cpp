/***************************************************
 * ESP32-Eyes for GC9A01 Round Display
 ***************************************************/

#include <TFT_eSPI.h>
#include "Face.h"

TFT_eSPI tft = TFT_eSPI();
Face *face;

void setup(void) {
  Serial.begin(115200);
  delay(500);
  Serial.println("ESP32 Eyes - GC9A01");

  tft.init();
  tft.setRotation(0);
  tft.fillScreen(TFT_BLACK);

  face = new Face(240, 240, 70);
  
  face->Expression.GoTo_Normal();
  face->RandomBehavior = true;
  face->Behavior.Timer.SetIntervalMillis(5000);
  face->RandomBlink = true;
  face->Blink.Timer.SetIntervalMillis(3000);
  face->RandomLook = true;
  face->Look.Timer.SetIntervalMillis(2000);

  face->Behavior.SetEmotion(eEmotions::Normal, 1.0);
  face->Behavior.SetEmotion(eEmotions::Happy, 0.8);
  face->Behavior.SetEmotion(eEmotions::Glee, 0.5);
}

void loop() {
  tft.fillScreen(TFT_BLACK);
  face->Update();
  delay(30);
}
