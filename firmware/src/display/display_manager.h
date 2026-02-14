#pragma once

#include "../config.h"
#include "lvgl.h"
#include "esp_lcd_panel_ops.h"
#include "freertos/FreeRTOS.h"
#include "freertos/semphr.h"

class DisplayManager {
public:
    bool begin();
    void setStatusText(const char* text);
    void setNotificationText(const char* text);
    void setButtonLabels(const char* btn1, const char* btn2,
                         const char* btn3, const char* btn4);
    void showIdleScreen();
    void showNotification(const char* text, const char* category);
    void setBrightness(uint8_t level);
    void update();

    // Must be called from any thread before touching LVGL objects
    bool lock(int timeout_ms = -1);
    void unlock();

private:
    void initPanel();
    void initLVGL();
    void initBacklight();
    void createUI();

    esp_lcd_panel_handle_t _panel = nullptr;
    lv_display_t* _disp = nullptr;
    SemaphoreHandle_t _lvglMux = nullptr;
    SemaphoreHandle_t _flushSem = nullptr;
    uint8_t* _rotBuf = nullptr;

    // LVGL UI objects
    lv_obj_t* _statusBar = nullptr;
    lv_obj_t* _statusLabel = nullptr;
    lv_obj_t* _notifLabel = nullptr;
    lv_obj_t* _btnObjs[4] = {};
    lv_obj_t* _btnLabels[4] = {};
};
