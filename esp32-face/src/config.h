/***************************************************
 * Koji Face Display Configuration
 * 
 * Copy this file to config_local.h and edit your settings.
 * config_local.h is gitignored so your credentials stay private.
 ***************************************************/

#ifndef KOJI_CONFIG_H
#define KOJI_CONFIG_H

// WiFi credentials
#ifndef WIFI_SSID
#define WIFI_SSID "your-wifi-ssid"
#endif

#ifndef WIFI_PASSWORD
#define WIFI_PASSWORD "your-wifi-password"
#endif

// Brain server URL
// This should be the IP/hostname of the machine running the brain server
#ifndef BRAIN_URL
#define BRAIN_URL "http://192.168.1.100:8080/api/state"
#endif

// How often to poll the brain server (milliseconds)
#ifndef POLL_INTERVAL_MS
#define POLL_INTERVAL_MS 500
#endif

// Display settings
#define SCREEN_WIDTH 240
#define SCREEN_HEIGHT 240
#define EYE_SIZE 70

#endif // KOJI_CONFIG_H
