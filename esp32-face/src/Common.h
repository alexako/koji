#ifndef COMMON_h
#define COMMON_h

#include <TFT_eSPI.h>

extern TFT_eSPI tft;

#ifdef USE_DOUBLE_BUFFER
// When double buffering, all drawing goes to this sprite
extern TFT_eSprite frameBuffer;
// Macro to redirect drawing calls to the sprite
#define DRAW_TARGET frameBuffer
#else
// Direct drawing to display
#define DRAW_TARGET tft
#endif

// Screen dimensions
#define SCREEN_WIDTH 240
#define SCREEN_HEIGHT 240

// Colors
#define BG_COLOR TFT_BLACK
#define EYE_COLOR TFT_CYAN

#endif
