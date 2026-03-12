/***************************************************
Copyright (c) 2023 Alastair Aitchison, Playful Technology
Adapted for TFT_eSPI / GC9A01
****************************************************/

#ifndef _EYEDRAWER_h
#define _EYEDRAWER_h

#include <Arduino.h>
#include "Common.h"
#include "EyeConfig.h"

enum CornerType {T_R, T_L, B_L, B_R};

class EyeDrawer {
  public:
    static void Draw(int16_t centerX, int16_t centerY, EyeConfig *config) {
      int32_t delta_y_top = config->Height * config->Slope_Top / 2.0;
      int32_t delta_y_bottom = config->Height * config->Slope_Bottom / 2.0;
      auto totalHeight = config->Height + delta_y_top - delta_y_bottom;
      
      if (config->Radius_Bottom > 0 && config->Radius_Top > 0 && totalHeight - 1 < config->Radius_Bottom + config->Radius_Top) {
        int32_t corrected_radius_top = (float)config->Radius_Top * (totalHeight - 1) / (config->Radius_Bottom + config->Radius_Top);
        int32_t corrected_radius_bottom = (float)config->Radius_Bottom * (totalHeight - 1) / (config->Radius_Bottom + config->Radius_Top);
        config->Radius_Top = corrected_radius_top;
        config->Radius_Bottom = corrected_radius_bottom;
      }

      int32_t TLc_y = centerY + config->OffsetY - config->Height/2 + config->Radius_Top - delta_y_top;
      int32_t TLc_x = centerX + config->OffsetX - config->Width/2 + config->Radius_Top;
      int32_t TRc_y = centerY + config->OffsetY - config->Height/2 + config->Radius_Top + delta_y_top;
      int32_t TRc_x = centerX + config->OffsetX + config->Width/2 - config->Radius_Top;
      int32_t BLc_y = centerY + config->OffsetY + config->Height/2 - config->Radius_Bottom - delta_y_bottom;
      int32_t BLc_x = centerX + config->OffsetX - config->Width/2 + config->Radius_Bottom;
      int32_t BRc_y = centerY + config->OffsetY + config->Height/2 - config->Radius_Bottom + delta_y_bottom;
      int32_t BRc_x = centerX + config->OffsetX + config->Width/2 - config->Radius_Bottom;
        
      int32_t min_c_x = min(TLc_x, BLc_x);
      int32_t max_c_x = max(TRc_x, BRc_x);
      int32_t min_c_y = min(TLc_y, TRc_y);
      int32_t max_c_y = max(BLc_y, BRc_y);

      // Fill eye centre
      FillRectangle(min_c_x, min_c_y, max_c_x, max_c_y, EYE_COLOR);

      // Fill eye outwards to meet edges of rounded corners 
      FillRectangle(TRc_x, TRc_y, BRc_x + config->Radius_Bottom, BRc_y, EYE_COLOR);
      FillRectangle(TLc_x - config->Radius_Top, TLc_y, BLc_x, BLc_y, EYE_COLOR);
      FillRectangle(TLc_x, TLc_y - config->Radius_Top, TRc_x, TRc_y, EYE_COLOR);
      FillRectangle(BLc_x, BLc_y, BRc_x, BRc_y + config->Radius_Bottom, EYE_COLOR);
        
      // Draw slanted edges
      if(config->Slope_Top > 0) {
        FillRectangularTriangle(TLc_x, TLc_y-config->Radius_Top, TRc_x, TRc_y-config->Radius_Top, BG_COLOR);
        FillRectangularTriangle(TRc_x, TRc_y-config->Radius_Top, TLc_x, TLc_y-config->Radius_Top, EYE_COLOR);
      } 
      else if(config->Slope_Top < 0) {
        FillRectangularTriangle(TRc_x, TRc_y-config->Radius_Top, TLc_x, TLc_y-config->Radius_Top, BG_COLOR);
        FillRectangularTriangle(TLc_x, TLc_y-config->Radius_Top, TRc_x, TRc_y-config->Radius_Top, EYE_COLOR);
      }

      if(config->Slope_Bottom > 0) {
        FillRectangularTriangle(BRc_x+config->Radius_Bottom, BRc_y+config->Radius_Bottom, BLc_x-config->Radius_Bottom, BLc_y+config->Radius_Bottom, BG_COLOR);
        FillRectangularTriangle(BLc_x-config->Radius_Bottom, BLc_y+config->Radius_Bottom, BRc_x+config->Radius_Bottom, BRc_y+config->Radius_Bottom, EYE_COLOR);
      }
      else if (config->Slope_Bottom < 0) {
        FillRectangularTriangle(BLc_x-config->Radius_Bottom, BLc_y+config->Radius_Bottom, BRc_x+config->Radius_Bottom, BRc_y+config->Radius_Bottom, BG_COLOR);
        FillRectangularTriangle(BRc_x+config->Radius_Bottom, BRc_y+config->Radius_Bottom, BLc_x-config->Radius_Bottom, BLc_y+config->Radius_Bottom, EYE_COLOR);
      }

      // Draw corners
      if(config->Radius_Top > 0) {
        FillEllipseCorner(T_L, TLc_x, TLc_y, config->Radius_Top, config->Radius_Top, EYE_COLOR);
        FillEllipseCorner(T_R, TRc_x, TRc_y, config->Radius_Top, config->Radius_Top, EYE_COLOR);
      }
      if(config->Radius_Bottom > 0) {
        FillEllipseCorner(B_L, BLc_x, BLc_y, config->Radius_Bottom, config->Radius_Bottom, EYE_COLOR);
        FillEllipseCorner(B_R, BRc_x, BRc_y, config->Radius_Bottom, config->Radius_Bottom, EYE_COLOR);
      }
    }

    static void FillEllipseCorner(CornerType corner, int16_t x0, int16_t y0, int32_t rx, int32_t ry, uint16_t color) {
      if (rx < 2) return;
      if (ry < 2) return;
      int32_t x, y;
      int32_t rx2 = rx * rx;
      int32_t ry2 = ry * ry;
      int32_t fx2 = 4 * rx2;
      int32_t fy2 = 4 * ry2;
      int32_t s;

      if (corner == T_R) {
        for(x = 0, y = ry, s = 2 * ry2 + rx2 * (1 - 2 * ry); ry2 * x <= rx2 * y; x++) {
          DRAW_TARGET.drawFastHLine(x0, y0 - y, x, color);
          if(s >= 0) { s += fx2 * (1 - y); y--; }
          s += ry2 * ((4 * x) + 6);
        }         
        for(x = rx, y = 0, s = 2 * rx2 + ry2 * (1 - 2 * rx); rx2 * y <= ry2 * x; y++) {
          DRAW_TARGET.drawFastHLine(x0, y0 - y, x, color);
          if (s >= 0) { s += fy2 * (1 - x); x--; }
          s += rx2 * ((4 * y) + 6);
        }
      }
      else if (corner == B_R) {
        for (x = 0, y = ry, s = 2 * ry2 + rx2 * (1 - 2 * ry); ry2 * x <= rx2 * y; x++) {
          DRAW_TARGET.drawFastHLine(x0, y0 + y - 1, x, color);
          if (s >= 0) { s += fx2 * (1 - y); y--; }
          s += ry2 * ((4 * x) + 6);
        }
        for (x = rx, y = 0, s = 2 * rx2 + ry2 * (1 - 2 * rx); rx2 * y <= ry2 * x; y++) {
          DRAW_TARGET.drawFastHLine(x0, y0 + y - 1, x, color);
          if (s >= 0) { s += fy2 * (1 - x); x--; }
          s += rx2 * ((4 * y) + 6);
        }
      }
      else if (corner == T_L) {
        for (x = 0, y = ry, s = 2 * ry2 + rx2 * (1 - 2 * ry); ry2 * x <= rx2 * y; x++) {
          DRAW_TARGET.drawFastHLine(x0 - x, y0 - y, x, color);
          if (s >= 0) { s += fx2 * (1 - y); y--; }
          s += ry2 * ((4 * x) + 6);
        }
        for (x = rx, y = 0, s = 2 * rx2 + ry2 * (1 - 2 * rx); rx2 * y <= ry2 * x; y++) {
          DRAW_TARGET.drawFastHLine(x0 - x, y0 - y, x, color);
          if (s >= 0) { s += fy2 * (1 - x); x--; }
          s += rx2 * ((4 * y) + 6);
        }
      }
      else if (corner == B_L) {
        for (x = 0, y = ry, s = 2 * ry2 + rx2 * (1 - 2 * ry); ry2 * x <= rx2 * y; x++) {
          DRAW_TARGET.drawFastHLine(x0 - x, y0 + y - 1, x, color);
          if (s >= 0) { s += fx2 * (1 - y); y--; }
          s += ry2 * ((4 * x) + 6);
        }
        for (x = rx, y = 0, s = 2 * rx2 + ry2 * (1 - 2 * rx); rx2 * y <= ry2 * x; y++) {
          DRAW_TARGET.drawFastHLine(x0 - x, y0 + y, x, color);
          if (s >= 0) { s += fy2 * (1 - x); x--; }
          s += rx2 * ((4 * y) + 6);
        }
      }
    }

    static void FillRectangle(int32_t x0, int32_t y0, int32_t x1, int32_t y1, uint16_t color) {
      int32_t l = min(x0, x1);
      int32_t r = max(x0, x1);
      int32_t t = min(y0, y1);
      int32_t b = max(y0, y1);
      int32_t w = r - l;
      int32_t h = b - t; 
      DRAW_TARGET.fillRect(l, t, w, h, color);
    }

    static void FillRectangularTriangle(int32_t x0, int32_t y0, int32_t x1, int32_t y1, uint16_t color) {
      DRAW_TARGET.fillTriangle(x0, y0, x1, y1, x1, y0, color);
    }

    static void FillTriangle(int32_t x0, int32_t y0, int32_t x1, int32_t y1, int32_t x2, int32_t y2, uint16_t color) {
      DRAW_TARGET.fillTriangle(x0, y0, x1, y1, x2, y2, color);
    }
};

#endif
