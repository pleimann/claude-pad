#include <Arduino.h>
#include "display_manager.h"
#include "display_config.h"
#include <cstring>
#include <esp_heap_caps.h>
#include <esp_lcd_panel_rgb.h>
#include <esp_timer.h>
#include <driver/ledc.h>
#include "vendor/st7701_bsp/esp_lcd_st7701.h"
#include "vendor/io_additions/esp_lcd_panel_io_additions.h"

// --- LVGL tick and task config ---
#define LVGL_TICK_PERIOD_MS    2
#define LVGL_TASK_MAX_DELAY_MS 500
#define LVGL_TASK_MIN_DELAY_MS 1
#define LVGL_TASK_STACK_SIZE   (8 * 1024)
#define LVGL_TASK_PRIORITY     5

#define BYTES_PER_PIXEL 2  // RGB565
#define BUFF_SIZE (LCD_H_RES * LCD_V_RES * BYTES_PER_PIXEL)

// --- Static references for C callbacks ---
static SemaphoreHandle_t s_flushSem = nullptr;
static SemaphoreHandle_t s_lvglMux = nullptr;
static uint8_t* s_rotBuf = nullptr;

// --- ISR: bounce frame finished ---
IRAM_ATTR static bool on_bounce_frame_finish(esp_lcd_panel_handle_t panel,
                                              const esp_lcd_rgb_panel_event_data_t* edata,
                                              void* user_ctx) {
    BaseType_t high_task_awoken = pdFALSE;
    xSemaphoreGiveFromISR(s_flushSem, &high_task_awoken);
    return high_task_awoken == pdTRUE;
}

// --- LVGL flush callback (with software rotation) ---
static void lvgl_flush_cb(lv_display_t* disp, const lv_area_t* area, uint8_t* color_p) {
    esp_lcd_panel_handle_t panel = (esp_lcd_panel_handle_t)lv_display_get_user_data(disp);
    lv_display_rotation_t rotation = lv_display_get_rotation(disp);

    if (rotation != LV_DISPLAY_ROTATION_0 && s_rotBuf) {
        lv_color_format_t cf = lv_display_get_color_format(disp);
        lv_area_t rotated_area = *area;
        lv_display_rotate_area(disp, &rotated_area);

        uint32_t src_stride = lv_draw_buf_width_to_stride(lv_area_get_width(area), cf);
        uint32_t dest_stride = lv_draw_buf_width_to_stride(lv_area_get_width(&rotated_area), cf);

        int32_t src_w = lv_area_get_width(area);
        int32_t src_h = lv_area_get_height(area);
        lv_draw_sw_rotate(color_p, s_rotBuf, src_w, src_h, src_stride, dest_stride, rotation, cf);

        esp_lcd_panel_draw_bitmap(panel, rotated_area.x1, rotated_area.y1,
                                  rotated_area.x2 + 1, rotated_area.y2 + 1, s_rotBuf);
    } else {
        esp_lcd_panel_draw_bitmap(panel, area->x1, area->y1,
                                  area->x2 + 1, area->y2 + 1, color_p);
    }
}

// --- LVGL flush wait callback ---
static void lvgl_flush_wait_cb(lv_display_t* disp) {
    xSemaphoreTake(s_flushSem, portMAX_DELAY);
}

// --- LVGL tick timer ---
static void lvgl_tick_cb(void* arg) {
    lv_tick_inc(LVGL_TICK_PERIOD_MS);
}

// --- LVGL task ---
static void lvgl_task(void* arg) {
    uint32_t task_delay_ms = LVGL_TASK_MAX_DELAY_MS;
    for (;;) {
        if (xSemaphoreTake(s_lvglMux, portMAX_DELAY) == pdTRUE) {
            task_delay_ms = lv_timer_handler();
            xSemaphoreGive(s_lvglMux);
        }
        if (task_delay_ms > LVGL_TASK_MAX_DELAY_MS) task_delay_ms = LVGL_TASK_MAX_DELAY_MS;
        else if (task_delay_ms < LVGL_TASK_MIN_DELAY_MS) task_delay_ms = LVGL_TASK_MIN_DELAY_MS;
        vTaskDelay(pdMS_TO_TICKS(task_delay_ms));
    }
}

// --- Backlight (PWM via LEDC) ---
void DisplayManager::initBacklight() {
    ledc_timer_config_t timer_conf = {};
    timer_conf.speed_mode = LEDC_LOW_SPEED_MODE;
    timer_conf.duty_resolution = LEDC_TIMER_8_BIT;
    timer_conf.timer_num = LEDC_TIMER_1;
    timer_conf.freq_hz = 50000;
    timer_conf.clk_cfg = LEDC_AUTO_CLK;
    ledc_timer_config(&timer_conf);

    ledc_channel_config_t channel_conf = {};
    channel_conf.gpio_num = PIN_LCD_BL;
    channel_conf.speed_mode = LEDC_LOW_SPEED_MODE;
    channel_conf.channel = LEDC_CHANNEL_1;
    channel_conf.timer_sel = LEDC_TIMER_1;
    channel_conf.duty = 255;  // Start with backlight off (inverted)
    channel_conf.hpoint = 0;
    ledc_channel_config(&channel_conf);
}

void DisplayManager::setBrightness(uint8_t level) {
    uint32_t duty = 255 - level;
    ledc_set_duty(LEDC_LOW_SPEED_MODE, LEDC_CHANNEL_1, duty);
    ledc_update_duty(LEDC_LOW_SPEED_MODE, LEDC_CHANNEL_1);
}

// --- RGB Panel init ---
void DisplayManager::initPanel() {
    // Create 3-wire SPI IO for ST7701 init commands
    spi_line_config_t line_config = {
        .cs_io_type = IO_TYPE_GPIO,
        .cs_gpio_num = PIN_LCD_SPI_CS,
        .scl_io_type = IO_TYPE_GPIO,
        .scl_gpio_num = PIN_LCD_SPI_SCK,
        .sda_io_type = IO_TYPE_GPIO,
        .sda_gpio_num = PIN_LCD_SPI_SDO,
        .io_expander = NULL,
    };
    esp_lcd_panel_io_3wire_spi_config_t io_config = ST7701_PANEL_IO_3WIRE_SPI_CONFIG(line_config, 0);
    esp_lcd_panel_io_handle_t io_handle = NULL;
    ESP_ERROR_CHECK(esp_lcd_new_panel_io_3wire_spi(&io_config, &io_handle));

    // RGB panel config with bounce buffers
    esp_lcd_rgb_panel_config_t rgb_config = {};
    rgb_config.clk_src = LCD_CLK_SRC_DEFAULT;
    rgb_config.psram_trans_align = 64;
    rgb_config.bounce_buffer_size_px = 10 * LCD_H_RES;
    rgb_config.num_fbs = 2;
    rgb_config.data_width = 16;
    rgb_config.bits_per_pixel = 16;
    rgb_config.de_gpio_num = PIN_LCD_DE;
    rgb_config.pclk_gpio_num = PIN_LCD_PCLK;
    rgb_config.vsync_gpio_num = PIN_LCD_VSYNC;
    rgb_config.hsync_gpio_num = PIN_LCD_HSYNC;
    rgb_config.disp_gpio_num = -1;
    rgb_config.flags.fb_in_psram = true;

    // Data pins: BGR order
    rgb_config.data_gpio_nums[0]  = PIN_LCD_B0;
    rgb_config.data_gpio_nums[1]  = PIN_LCD_B1;
    rgb_config.data_gpio_nums[2]  = PIN_LCD_B2;
    rgb_config.data_gpio_nums[3]  = PIN_LCD_B3;
    rgb_config.data_gpio_nums[4]  = PIN_LCD_B4;
    rgb_config.data_gpio_nums[5]  = PIN_LCD_G0;
    rgb_config.data_gpio_nums[6]  = PIN_LCD_G1;
    rgb_config.data_gpio_nums[7]  = PIN_LCD_G2;
    rgb_config.data_gpio_nums[8]  = PIN_LCD_G3;
    rgb_config.data_gpio_nums[9]  = PIN_LCD_G4;
    rgb_config.data_gpio_nums[10] = PIN_LCD_G5;
    rgb_config.data_gpio_nums[11] = PIN_LCD_R0;
    rgb_config.data_gpio_nums[12] = PIN_LCD_R1;
    rgb_config.data_gpio_nums[13] = PIN_LCD_R2;
    rgb_config.data_gpio_nums[14] = PIN_LCD_R3;
    rgb_config.data_gpio_nums[15] = PIN_LCD_R4;

    rgb_config.timings.pclk_hz           = LCD_PCLK_HZ;
    rgb_config.timings.h_res             = LCD_H_RES;
    rgb_config.timings.v_res             = LCD_V_RES;
    rgb_config.timings.hsync_back_porch  = LCD_HSYNC_BACK_PORCH;
    rgb_config.timings.hsync_front_porch = LCD_HSYNC_FRONT_PORCH;
    rgb_config.timings.hsync_pulse_width = LCD_HSYNC_PULSE_WIDTH;
    rgb_config.timings.vsync_back_porch  = LCD_VSYNC_BACK_PORCH;
    rgb_config.timings.vsync_front_porch = LCD_VSYNC_FRONT_PORCH;
    rgb_config.timings.vsync_pulse_width = LCD_VSYNC_PULSE_WIDTH;

    // ST7701 vendor config
    st7701_vendor_config_t vendor_config = {};
    vendor_config.rgb_config = &rgb_config;
    vendor_config.init_cmds = lcd_init_cmds;
    vendor_config.init_cmds_size = sizeof(lcd_init_cmds) / sizeof(st7701_lcd_init_cmd_t);
    vendor_config.flags.mirror_by_cmd = 1;
    vendor_config.flags.enable_io_multiplex = 0;

    const esp_lcd_panel_dev_config_t panel_config = {
        .reset_gpio_num = PIN_LCD_RESET,
        .rgb_ele_order = LCD_RGB_ELEMENT_ORDER_RGB,
        .bits_per_pixel = 16,
        .vendor_config = &vendor_config,
    };

    ESP_ERROR_CHECK(esp_lcd_new_panel_st7701(io_handle, &panel_config, &_panel));

    // Register bounce-frame-finish ISR
    esp_lcd_rgb_panel_event_callbacks_t cbs = {
        .on_bounce_frame_finish = on_bounce_frame_finish,
    };
    ESP_ERROR_CHECK(esp_lcd_rgb_panel_register_event_callbacks(_panel, &cbs, NULL));

    ESP_ERROR_CHECK(esp_lcd_panel_reset(_panel));
    ESP_ERROR_CHECK(esp_lcd_panel_init(_panel));

    Serial.println("[display] RGB panel created with bounce buffers");
}

// --- LVGL init ---
void DisplayManager::initLVGL() {
    lv_init();

    // Create display (native portrait resolution)
    _disp = lv_display_create(LCD_H_RES, LCD_V_RES);
    lv_display_set_flush_cb(_disp, lvgl_flush_cb);
    lv_display_set_flush_wait_cb(_disp, lvgl_flush_wait_cb);

    // Allocate LVGL render buffers in PSRAM
    uint8_t* buf1 = (uint8_t*)heap_caps_malloc(BUFF_SIZE, MALLOC_CAP_SPIRAM);
    uint8_t* buf2 = (uint8_t*)heap_caps_malloc(BUFF_SIZE, MALLOC_CAP_SPIRAM);
    if (!buf1 || !buf2) {
        Serial.println("[display] ERROR: LVGL buffer allocation failed!");
        return;
    }
    lv_display_set_buffers(_disp, buf1, buf2, BUFF_SIZE, LV_DISPLAY_RENDER_MODE_PARTIAL);
    lv_display_set_user_data(_disp, _panel);

    // Software rotation: 90 degrees for landscape (820x320)
    _rotBuf = (uint8_t*)heap_caps_malloc(BUFF_SIZE, MALLOC_CAP_SPIRAM);
    s_rotBuf = _rotBuf;
    lv_display_set_rotation(_disp, LV_DISPLAY_ROTATION_90);

    // LVGL tick timer (2ms)
    const esp_timer_create_args_t tick_args = {
        .callback = &lvgl_tick_cb,
        .name = "lvgl_tick"
    };
    esp_timer_handle_t tick_timer = NULL;
    ESP_ERROR_CHECK(esp_timer_create(&tick_args, &tick_timer));
    ESP_ERROR_CHECK(esp_timer_start_periodic(tick_timer, LVGL_TICK_PERIOD_MS * 1000));

    // LVGL task on core 1
    xTaskCreatePinnedToCore(lvgl_task, "LVGL", LVGL_TASK_STACK_SIZE, NULL, LVGL_TASK_PRIORITY, NULL, 1);

    Serial.println("[display] LVGL initialized (820x320 landscape)");
}

// --- UI creation ---
void DisplayManager::createUI() {
    lv_obj_t* scr = lv_display_get_screen_active(_disp);
    lv_obj_set_style_bg_color(scr, lv_color_hex(0x10141a), 0);

    // Status bar (top 30px)
    _statusBar = lv_obj_create(scr);
    lv_obj_set_size(_statusBar, SCREEN_WIDTH, 30);
    lv_obj_set_pos(_statusBar, 0, 0);
    lv_obj_set_style_bg_color(_statusBar, lv_color_hex(0x1a2030), 0);
    lv_obj_set_style_radius(_statusBar, 0, 0);
    lv_obj_set_style_border_width(_statusBar, 0, 0);
    lv_obj_set_style_pad_all(_statusBar, 0, 0);

    _statusLabel = lv_label_create(_statusBar);
    lv_label_set_text(_statusLabel, "Ready");
    lv_obj_set_style_text_color(_statusLabel, lv_color_hex(0x00ff00), 0);
    lv_obj_set_style_text_font(_statusLabel, FONT_STATUS, 0);
    lv_obj_align(_statusLabel, LV_ALIGN_LEFT_MID, 8, 0);

    // Notification text area (middle)
    _notifLabel = lv_label_create(scr);
    lv_label_set_text(_notifLabel, "");
    lv_label_set_long_mode(_notifLabel, LV_LABEL_LONG_WRAP);
    lv_obj_set_width(_notifLabel, SCREEN_WIDTH - 16);
    lv_obj_set_pos(_notifLabel, 8, 38);
    lv_obj_set_style_text_color(_notifLabel, lv_color_hex(0xffffff), 0);
    lv_obj_set_style_text_font(_notifLabel, FONT_NOTIF, 0);

    // Button bar (bottom 70px)
    int btnWidth = SCREEN_WIDTH / 4;
    for (int i = 0; i < 4; i++) {
        _btnObjs[i] = lv_button_create(scr);
        lv_obj_set_size(_btnObjs[i], btnWidth - 8, 62);
        lv_obj_set_pos(_btnObjs[i], i * btnWidth + 4, SCREEN_HEIGHT - 66);
        lv_obj_set_style_bg_color(_btnObjs[i], lv_color_hex(0x2a3040), 0);
        lv_obj_set_style_radius(_btnObjs[i], 6, 0);

        _btnLabels[i] = lv_label_create(_btnObjs[i]);
        char label[2] = {(char)('1' + i), '\0'};
        lv_label_set_text(_btnLabels[i], label);
        lv_obj_set_style_text_color(_btnLabels[i], lv_color_hex(0xffffff), 0);
        lv_obj_set_style_text_font(_btnLabels[i], FONT_BUTTON, 0);
        lv_obj_center(_btnLabels[i]);
    }
}

// --- Public API ---
bool DisplayManager::begin() {
    size_t psram_free = heap_caps_get_free_size(MALLOC_CAP_SPIRAM);
    Serial.printf("[display] PSRAM free: %u bytes\n", psram_free);
    if (psram_free == 0) {
        Serial.println("[display] WARNING: No PSRAM detected!");
        return false;
    }

    _lvglMux = xSemaphoreCreateMutex();
    _flushSem = xSemaphoreCreateBinary();
    s_lvglMux = _lvglMux;
    s_flushSem = _flushSem;

    initBacklight();
    initPanel();
    initLVGL();

    if (lock()) {
        createUI();
        unlock();
    }

    setBrightness(200);
    return true;
}

bool DisplayManager::lock(int timeout_ms) {
    const TickType_t ticks = (timeout_ms == -1) ? portMAX_DELAY : pdMS_TO_TICKS(timeout_ms);
    return xSemaphoreTake(_lvglMux, ticks) == pdTRUE;
}

void DisplayManager::unlock() {
    xSemaphoreGive(_lvglMux);
}

void DisplayManager::setStatusText(const char* text) {
    if (lock()) {
        lv_label_set_text(_statusLabel, text);
        unlock();
    }
}

void DisplayManager::setNotificationText(const char* text) {
    if (lock()) {
        lv_label_set_text(_notifLabel, text);
        unlock();
    }
}

void DisplayManager::setButtonLabels(const char* btn1, const char* btn2,
                                     const char* btn3, const char* btn4) {
    const char* labels[] = {btn1, btn2, btn3, btn4};
    if (lock()) {
        for (int i = 0; i < 4; i++) {
            if (labels[i]) {
                lv_label_set_text(_btnLabels[i], labels[i]);
            }
        }
        unlock();
    }
}

void DisplayManager::showIdleScreen() {
    setStatusText("Waiting for connection...");
    setNotificationText("");
    setButtonLabels("1", "2", "3", "4");
}

void DisplayManager::showNotification(const char* text, const char* category) {
    if (category && category[0]) {
        char statusBuf[128];
        snprintf(statusBuf, sizeof(statusBuf), "[%s]", category);
        setStatusText(statusBuf);
    }
    setNotificationText(text);
}

void DisplayManager::update() {
    // LVGL handles rendering automatically via its task
    // No manual update needed
}
