#pragma once

#include <cstdint>

// ----- I2C (Seesaw) -----
#define PIN_I2C_SDA 15
#define PIN_I2C_SCL 7
#define SEESAW_I2C_ADDR 0x49

// Seesaw button pins (on the ATtiny, not ESP32 GPIOs)
#define SEESAW_BTN_1 1
#define SEESAW_BTN_2 2
#define SEESAW_BTN_3 3
#define SEESAW_BTN_4 4
#define SEESAW_NEOPIX_PIN 0
#define SEESAW_NEOPIXEL_COUNT 4

// ----- Display: 3-Wire SPI (ST7701 init) -----
#define PIN_LCD_SPI_CS 0
#define PIN_LCD_SPI_SCK 2
#define PIN_LCD_SPI_SDO 1

// ----- Display: RGB Parallel Data -----
#define PIN_LCD_DE 40
#define PIN_LCD_PCLK 41
#define PIN_LCD_VSYNC 39
#define PIN_LCD_HSYNC 38
#define PIN_LCD_RESET 16

// Red: R0-R4
#define PIN_LCD_R0 17
#define PIN_LCD_R1 46
#define PIN_LCD_R2 3
#define PIN_LCD_R3 8
#define PIN_LCD_R4 18

// Green: G0-G5
#define PIN_LCD_G0 14
#define PIN_LCD_G1 13
#define PIN_LCD_G2 12
#define PIN_LCD_G3 11
#define PIN_LCD_G4 10
#define PIN_LCD_G5 9

// Blue: B0-B4
#define PIN_LCD_B0 21
#define PIN_LCD_B1 5
#define PIN_LCD_B2 45
#define PIN_LCD_B3 48
#define PIN_LCD_B4 47

// ----- Backlight -----
#define PIN_LCD_BL 6

// ----- Display Dimensions -----
#define LCD_H_RES 320
#define LCD_V_RES 820
#define LCD_PCLK_HZ (18 * 1000 * 1000)

// RGB Timing
#define LCD_HSYNC_BACK_PORCH 30
#define LCD_HSYNC_FRONT_PORCH 30
#define LCD_HSYNC_PULSE_WIDTH 6
#define LCD_VSYNC_BACK_PORCH 20
#define LCD_VSYNC_FRONT_PORCH 20
#define LCD_VSYNC_PULSE_WIDTH 40

// After 90-degree rotation
#define SCREEN_WIDTH 820
#define SCREEN_HEIGHT 320

// ----- Communication Protocol -----
#define MSG_DISPLAY_TEXT 0x01
#define MSG_BUTTON 0x02
#define MSG_SET_LEDS 0x03
#define MSG_STATUS 0x04
#define MSG_CLEAR 0x05
#define MSG_SET_LABELS 0x06
#define MSG_HEARTBEAT 0x07

#define FRAME_START_BYTE 0xAA
#define MAX_MSG_LEN 512
#define SERIAL_BAUD 115200
