/*
 * Minimal ESP-IDF + LwIP bridge for picoceci on ESP32-S3.
 *
 * This file is compiled via target extra-files and provides stable C symbols
 * callable from Go/TinyGo to initialize WiFi and expose TCP sockets.
 */

#include <stdint.h>
#include <stdbool.h>
#include <string.h>

typedef int32_t esp_err_t;
typedef struct esp_netif_s esp_netif_t;

/* Minimal ESP-IDF constants used by this bridge. */
#define ESP_OK 0
#define ESP_ERR_INVALID_STATE 0x103
#define ESP_ERR_NVS_NO_FREE_PAGES 0x110d
#define ESP_ERR_NVS_NEW_VERSION_FOUND 0x1110
#define ESP_ERR_WIFI_NOT_CONNECT 0x3006

#define WIFI_MODE_STA 1
#define WIFI_IF_STA 0
#define WIFI_AUTH_WPA2_PSK 3

#define AF_INET 2
#define SOCK_STREAM 1
#define IPPROTO_TCP 6
#define SOL_SOCKET 0xfff
#define SO_REUSEADDR 0x0004
#define INADDR_ANY 0

#define pdMS_TO_TICKS(ms) (ms)

/* ABI-compatible subset of ESP-IDF structs. */
typedef struct
{
    uint32_t mode;
    int8_t rssi;
} auth_mode_threshold_t;

typedef struct
{
    bool capable;
    bool required;
} pmf_cfg_t;

typedef struct
{
    uint8_t ssid[32];
    uint8_t password[64];
    uint32_t scan_method;
    bool bssid_set;
    uint8_t bssid[6];
    uint8_t channel;
    uint16_t listen_interval;
    uint32_t sort_method;
    auth_mode_threshold_t threshold;
    pmf_cfg_t pmf_cfg;
    uint8_t sae_h2e_identifier[32];
} wifi_sta_config_t;

typedef union
{
    wifi_sta_config_t sta;
} wifi_config_t;

typedef struct
{
    uint32_t ip;
    uint32_t netmask;
    uint32_t gw;
} esp_netif_ip_info_t;

typedef struct
{
    uint16_t sin_family;
    uint16_t sin_port;
    uint32_t sin_addr;
    uint8_t sin_zero[8];
} sockaddr_in_t;

/* External functions provided by ESP-IDF/TinyGo link. */
extern esp_err_t nvs_flash_init(void);
extern esp_err_t nvs_flash_erase(void);
extern esp_err_t esp_netif_init(void);
extern esp_err_t esp_event_loop_create_default(void);
extern esp_netif_t *esp_netif_create_default_wifi_sta(void);
extern esp_err_t esp_netif_get_ip_info(esp_netif_t *netif, esp_netif_ip_info_t *ip_info);
extern esp_err_t canal_wifi_init_default(void);
extern esp_err_t esp_wifi_set_mode(uint32_t mode);
extern esp_err_t esp_wifi_set_config(uint32_t interface_, wifi_config_t *conf);
extern esp_err_t esp_wifi_start(void);
extern esp_err_t esp_wifi_connect(void);
extern esp_err_t esp_wifi_disconnect(void);
extern void vTaskDelay(uint32_t ticks);

extern int32_t lwip_socket(int32_t domain, int32_t typ, int32_t protocol);
extern int32_t lwip_bind(int32_t s, const void *name, uint32_t namelen);
extern int32_t lwip_listen(int32_t s, int32_t backlog);
extern int32_t lwip_accept(int32_t s, void *addr, void *addrlen);
extern int32_t lwip_recv(int32_t s, void *mem, int32_t len, int32_t flags);
extern int32_t lwip_send(int32_t s, const void *data, int32_t size, int32_t flags);
extern int32_t lwip_close(int32_t s);
extern int32_t lwip_setsockopt(int32_t s, int32_t level, int32_t optname, const void *optval, uint32_t optlen);

static uint16_t htons16(uint16_t v) { return (uint16_t)((v << 8) | (v >> 8)); }
static uint16_t ntohs16(uint16_t v) { return (uint16_t)((v << 8) | (v >> 8)); }
static uint32_t ntohl32(uint32_t v)
{
    return ((v & 0x000000FFU) << 24) |
           ((v & 0x0000FF00U) << 8) |
           ((v & 0x00FF0000U) >> 8) |
           ((v & 0xFF000000U) >> 24);
}
static uint32_t htonl32(uint32_t v) { return ntohl32(v); }

static bool bridge_ready = false;
static esp_netif_t *bridge_sta = NULL;
static bool bridge_wifi_started = false;

/*
 * Contract:
 * - Initializes NVS, netif, event loop, and default WiFi STA netif.
 * - Idempotent for repeated calls once initialized.
 * Returns:
 * - 0 on success
 * - negative bridge code on failure (-1..-6)
 */
int32_t picoceci_bridge_wifi_stack_init(void)
{
    esp_err_t err;

    err = nvs_flash_init();
    if (err != ESP_OK)
    {
        if (err == ESP_ERR_NVS_NO_FREE_PAGES || err == ESP_ERR_NVS_NEW_VERSION_FOUND)
        {
            if (nvs_flash_erase() != ESP_OK)
            {
                return -1;
            }
            err = nvs_flash_init();
        }
    }
    if (err != ESP_OK)
    {
        return -2;
    }

    err = esp_netif_init();
    if (err != ESP_OK && err != ESP_ERR_INVALID_STATE)
    {
        return -3;
    }

    err = esp_event_loop_create_default();
    if (err != ESP_OK && err != ESP_ERR_INVALID_STATE)
    {
        return -4;
    }

    if (bridge_sta == NULL)
    {
        bridge_sta = esp_netif_create_default_wifi_sta();
        if (bridge_sta == NULL)
        {
            return -5;
        }
    }

    err = canal_wifi_init_default();
    if (err != ESP_OK && err != ESP_ERR_INVALID_STATE)
    {
        return -6;
    }

    bridge_ready = true;
    return 0;
}

/*
 * Contract:
 * - Connects STA using UTF-8 NUL-terminated ssid/password.
 * - password may be NULL for open networks.
 * - out_ip may be NULL; if non-NULL receives host-order IPv4.
 * - Blocks up to timeout_ms while polling for assigned IP.
 * Returns:
 * - 0 on success
 * - negative bridge code on failure (-10..-15)
 */
int32_t picoceci_bridge_wifi_connect(const char *ssid,
                                     const char *password,
                                     uint32_t timeout_ms,
                                     uint32_t *out_ip)
{
    if (!bridge_ready || bridge_sta == NULL || ssid == NULL || ssid[0] == '\0')
    {
        return -10;
    }

    esp_err_t err = esp_wifi_set_mode(WIFI_MODE_STA);
    if (err != ESP_OK)
    {
        return -11;
    }

    wifi_config_t cfg;
    memset(&cfg, 0, sizeof(cfg));
    strncpy((char *)cfg.sta.ssid, ssid, sizeof(cfg.sta.ssid) - 1);
    if (password != NULL)
    {
        strncpy((char *)cfg.sta.password, password, sizeof(cfg.sta.password) - 1);
    }
    cfg.sta.threshold.mode = WIFI_AUTH_WPA2_PSK;

    err = esp_wifi_set_config(WIFI_IF_STA, &cfg);
    if (err != ESP_OK)
    {
        return -12;
    }

    if (!bridge_wifi_started)
    {
        err = esp_wifi_start();
        if (err != ESP_OK)
        {
            return -13;
        }
        bridge_wifi_started = true;
    }

    err = esp_wifi_connect();
    if (err != ESP_OK)
    {
        return -14;
    }

    const uint32_t step_ms = 100;
    uint32_t waited = 0;
    while (waited < timeout_ms)
    {
        esp_netif_ip_info_t info;
        memset(&info, 0, sizeof(info));
        if (esp_netif_get_ip_info(bridge_sta, &info) == ESP_OK && info.ip != 0)
        {
            if (out_ip != NULL)
            {
                *out_ip = ntohl32(info.ip);
            }
            return 0;
        }
        vTaskDelay(pdMS_TO_TICKS(step_ms));
        waited += step_ms;
    }

    return -15;
}

/*
 * Contract:
 * - Best-effort STA disconnect; safe if not currently connected.
 * Returns:
 * - 0 on success
 * - negative bridge code on failure (-20..-21)
 */
int32_t picoceci_bridge_wifi_disconnect(void)
{
    if (!bridge_ready)
    {
        return -20;
    }
    esp_err_t err = esp_wifi_disconnect();
    if (err == ESP_OK || err == ESP_ERR_WIFI_NOT_CONNECT)
    {
        return 0;
    }
    return -21;
}

/*
 * Contract:
 * - Opens a TCP server socket and listens on the provided port.
 * - Caller owns returned fd and must close it with picoceci_bridge_tcp_close.
 * Returns:
 * - fd >= 0 on success
 * - negative bridge code on failure (-30..-32)
 */
int32_t picoceci_bridge_tcp_listen(uint16_t port)
{
    int fd = lwip_socket(AF_INET, SOCK_STREAM, IPPROTO_TCP);
    if (fd < 0)
    {
        return -30;
    }

    int opt = 1;
    (void)lwip_setsockopt(fd, SOL_SOCKET, SO_REUSEADDR, &opt, sizeof(opt));

    sockaddr_in_t addr;
    memset(&addr, 0, sizeof(addr));
    addr.sin_family = AF_INET;
    addr.sin_port = htons16(port);
    addr.sin_addr = htonl32(INADDR_ANY);

    if (lwip_bind(fd, &addr, sizeof(addr)) < 0)
    {
        lwip_close(fd);
        return -31;
    }

    if (lwip_listen(fd, 1) < 0)
    {
        lwip_close(fd);
        return -32;
    }

    return fd;
}

/*
 * Contract:
 * - Blocks waiting for one incoming client on server_fd.
 * - out_ip/out_port are optional; if non-NULL receive host-order values.
 * - Caller owns returned client fd and must close it.
 * Returns:
 * - client fd >= 0 on success
 * - -40 on accept failure
 */
int32_t picoceci_bridge_tcp_accept(int32_t server_fd, uint32_t *out_ip, uint16_t *out_port)
{
    sockaddr_in_t client;
    uint32_t clen = sizeof(client);
    int cfd = lwip_accept(server_fd, &client, &clen);
    if (cfd < 0)
    {
        return -40;
    }

    if (out_ip != NULL)
    {
        *out_ip = ntohl32(client.sin_addr);
    }
    if (out_port != NULL)
    {
        *out_port = ntohs16(client.sin_port);
    }

    return cfd;
}

/*
 * Contract:
 * - Receives up to len bytes into caller-owned buffer.
 * - buf may be NULL only when len == 0 (treated as no-op).
 * Returns lwip semantics:
 * - n > 0 bytes read
 * - 0 for connection closed/EOF
 * - negative on error
 */
int32_t picoceci_bridge_tcp_recv(int32_t fd, uint8_t *buf, uint32_t len)
{
    if (buf == NULL || len == 0)
    {
        return 0;
    }
    return lwip_recv(fd, buf, (int)len, 0);
}

/*
 * Contract:
 * - Sends up to len bytes from caller-owned buffer.
 * - buf may be NULL only when len == 0 (treated as no-op).
 * Returns lwip semantics:
 * - n > 0 bytes written
 * - 0 for no-op
 * - negative on error
 */
int32_t picoceci_bridge_tcp_send(int32_t fd, const uint8_t *buf, uint32_t len)
{
    if (buf == NULL || len == 0)
    {
        return 0;
    }
    return lwip_send(fd, buf, (int)len, 0);
}

/*
 * Contract:
 * - Closes any fd returned from listen/accept.
 * Returns lwip close result (0 success, negative error).
 */
int32_t picoceci_bridge_tcp_close(int32_t fd)
{
    return lwip_close(fd);
}
