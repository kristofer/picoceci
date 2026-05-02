/*
 * PSRAM initialization for ESP32-S3-N16R8 (8MB Octal PSRAM)
 *
 * This file initializes the octal PSRAM and configures the MMU to map it
 * at 0x3D000000 (after the 16MB flash region).
 *
 * IMPORTANT: This must run BEFORE the Go runtime starts using the heap.
 * It should be called from the boot sequence (call_start_cpu0).
 *
 * Based on ESP-IDF components/esp_psram/esp32s3/esp_psram_impl_octal.c
 *
 * Compile with:
 *   xtensa-esp32s3-elf-gcc -c -O2 -mlongcalls psram_init.c -o psram_init.o
 *
 * Link with TinyGo using ldflags:
 *   tinygo build -ldflags="-L. psram_init.o" ...
 */

#include <stdint.h>

/* ESP32-S3 Register addresses */
#define DR_REG_SPI_MEM_BASE(i)    (0x60002000 + (i) * 0x1000)
#define SPI_MEM_CTRL_REG(i)       (DR_REG_SPI_MEM_BASE(i) + 0x008)
#define SPI_MEM_CTRL1_REG(i)      (DR_REG_SPI_MEM_BASE(i) + 0x00C)
#define SPI_MEM_CTRL2_REG(i)      (DR_REG_SPI_MEM_BASE(i) + 0x010)
#define SPI_MEM_CACHE_FCTRL_REG(i) (DR_REG_SPI_MEM_BASE(i) + 0x03C)
#define SPI_MEM_SRAM_CMD_REG(i)   (DR_REG_SPI_MEM_BASE(i) + 0x044)
#define SPI_MEM_SRAM_DRD_CMD_REG(i) (DR_REG_SPI_MEM_BASE(i) + 0x048)
#define SPI_MEM_SRAM_DWR_CMD_REG(i) (DR_REG_SPI_MEM_BASE(i) + 0x04C)
#define SPI_MEM_SRAM_CLK_REG(i)   (DR_REG_SPI_MEM_BASE(i) + 0x050)
#define SPI_MEM_TIMING_CALI_REG(i) (DR_REG_SPI_MEM_BASE(i) + 0x0A8)

#define DR_REG_SYSTEM_BASE        0x600C0000
#define SYSTEM_PERIP_CLK_EN0_REG  (DR_REG_SYSTEM_BASE + 0x018)
#define SYSTEM_PERIP_RST_EN0_REG  (DR_REG_SYSTEM_BASE + 0x020)
#define SYSTEM_SYSCLK_CONF_REG    (DR_REG_SYSTEM_BASE + 0x058)

#define DR_REG_EXTMEM_BASE        0x600C4000
#define EXTMEM_DCACHE_CTRL_REG    (DR_REG_EXTMEM_BASE + 0x000)
#define EXTMEM_DCACHE_CTRL1_REG   (DR_REG_EXTMEM_BASE + 0x004)

/* MMU registers for mapping PSRAM */
#define DR_REG_MMU_TABLE          0x600C5000
#define MMU_PAGE_SIZE             0x10000  /* 64KB pages */
#define MMU_INVALID               (1 << 14)
#define MMU_ACCESS_SPIRAM         (1 << 15)
#define MMU_VALID                 0

/* PSRAM base virtual address (after 16MB flash) */
#define PSRAM_VADDR_START         0x3D000000
#define PSRAM_SIZE                (8 * 1024 * 1024)  /* 8MB */
#define FLASH_SIZE                (16 * 1024 * 1024) /* 16MB */

/* Helper macros */
#define REG_WRITE(addr, val)  (*(volatile uint32_t *)(addr) = (val))
#define REG_READ(addr)        (*(volatile uint32_t *)(addr))
#define REG_SET_BIT(addr, bit) REG_WRITE(addr, REG_READ(addr) | (bit))
#define REG_CLR_BIT(addr, bit) REG_WRITE(addr, REG_READ(addr) & ~(bit))

/* Octal PSRAM mode register commands */
#define PSRAM_CMD_READ            0x00
#define PSRAM_CMD_WRITE           0x80
#define PSRAM_CMD_READ_ID         0x9F
#define PSRAM_CMD_ENTER_QMODE     0x35
#define PSRAM_CMD_EXIT_QMODE      0xF5
#define PSRAM_CMD_READ_REG        0x40
#define PSRAM_CMD_WRITE_REG       0xC0

/* SPI1 is used for PSRAM on ESP32-S3 */
#define PSRAM_SPI_NUM             1

/* Forward declarations */
static void psram_gpio_config(void);
static void psram_set_cs_timing(void);
static int psram_enable_octal_mode(void);
static void psram_cache_init(void);

/*
 * Initialize PSRAM hardware and MMU mapping.
 * Returns 0 on success, -1 on failure.
 *
 * This function must be called early in the boot process,
 * before any heap allocations occur.
 */
int psram_init(void) __attribute__((section(".iram")));

int psram_init(void) {
    /* 1. Enable MSPI peripheral clock */
    REG_SET_BIT(SYSTEM_PERIP_CLK_EN0_REG, (1 << 6));  /* SPI2 clock */

    /* 2. Configure GPIO for octal PSRAM (GPIO 33-38) */
    psram_gpio_config();

    /* 3. Configure SPI timing */
    psram_set_cs_timing();

    /* 4. Enable octal PSRAM mode */
    if (psram_enable_octal_mode() != 0) {
        return -1;  /* PSRAM not detected or initialization failed */
    }

    /* 5. Configure cache and MMU to map PSRAM */
    psram_cache_init();

    return 0;
}

static void psram_gpio_config(void) {
    /* ESP32-S3-N16R8 uses these pins for octal PSRAM:
     * SPICS1 (GPIO 26) - CS for PSRAM
     * SPID4-D7 (GPIO 33-36) - Data lines 4-7 for octal mode
     * SPICLK (GPIO 30) - Clock (shared with flash)
     * SPID0-D3 (GPIO 27-29, 32) - Data lines 0-3 (shared with flash)
     *
     * These are typically already configured by the ROM bootloader
     * for octal flash mode, we just need to enable PSRAM CS.
     */

    /* GPIO 26 function select for SPI (SPICS1) */
    volatile uint32_t *gpio_func_out_sel = (volatile uint32_t *)0x60004554;
    *gpio_func_out_sel = 0x100;  /* SPI function */
}

static void psram_set_cs_timing(void) {
    /* Set PSRAM chip select timing
     * These values are for 80MHz SPI clock with octal PSRAM
     */
    uint32_t ctrl1 = REG_READ(SPI_MEM_CTRL1_REG(PSRAM_SPI_NUM));
    ctrl1 &= ~(0x1F << 0);  /* Clear CS hold time */
    ctrl1 |= (1 << 0);      /* CS hold: 1 cycle */
    ctrl1 &= ~(0x1F << 5);  /* Clear CS setup time */
    ctrl1 |= (1 << 5);      /* CS setup: 1 cycle */
    REG_WRITE(SPI_MEM_CTRL1_REG(PSRAM_SPI_NUM), ctrl1);
}

static int psram_enable_octal_mode(void) {
    /* Configure SPI for octal mode with PSRAM commands */

    /* Set octal mode in SRAM_CMD register */
    uint32_t sram_cmd = REG_READ(SPI_MEM_SRAM_CMD_REG(PSRAM_SPI_NUM));
    sram_cmd |= (1 << 5);   /* SRAM_OCT: enable octal mode */
    sram_cmd |= (1 << 0);   /* SRAM_WREN: write enable */
    REG_WRITE(SPI_MEM_SRAM_CMD_REG(PSRAM_SPI_NUM), sram_cmd);

    /* Configure read command for octal PSRAM */
    uint32_t drd_cmd = (0xEC << 0)   /* Read command (octal sync read) */
                     | (8 << 8)      /* Command bits */
                     | (32 << 16);   /* Address bits */
    REG_WRITE(SPI_MEM_SRAM_DRD_CMD_REG(PSRAM_SPI_NUM), drd_cmd);

    /* Configure write command for octal PSRAM */
    uint32_t dwr_cmd = (0x8C << 0)   /* Write command (octal sync write) */
                     | (8 << 8)      /* Command bits */
                     | (32 << 16);   /* Address bits */
    REG_WRITE(SPI_MEM_SRAM_DWR_CMD_REG(PSRAM_SPI_NUM), dwr_cmd);

    /* Configure clock divider (match flash speed) */
    uint32_t clk_reg = (1 << 0)      /* CLK_DIV: divide by 2 */
                     | (0 << 6)      /* CLK_EQU_SYSCLK: 0 */
                     | (1 << 12);    /* CLK_CNT_L */
    REG_WRITE(SPI_MEM_SRAM_CLK_REG(PSRAM_SPI_NUM), clk_reg);

    /* Brief delay for PSRAM to stabilize */
    for (volatile int i = 0; i < 1000; i++);

    /* TODO: Read PSRAM ID to verify it's present
     * For now, assume PSRAM is present on N16R8 boards
     */

    return 0;
}

static void psram_cache_init(void) {
    /* Disable D-cache temporarily */
    REG_CLR_BIT(EXTMEM_DCACHE_CTRL_REG, (1 << 0));

    /* Configure MMU to map PSRAM at PSRAM_VADDR_START
     *
     * The MMU table has one entry per 64KB page.
     * Flash is mapped at 0x3C000000 (256 entries for 16MB)
     * PSRAM should be mapped at 0x3D000000 (128 entries for 8MB)
     *
     * MMU entry format:
     * - Bits 0-13: Physical page number
     * - Bit 14: Invalid flag
     * - Bit 15: Access type (0=flash, 1=PSRAM)
     */

    volatile uint32_t *mmu_table = (volatile uint32_t *)DR_REG_MMU_TABLE;

    /* Calculate starting MMU entry for PSRAM region */
    uint32_t vaddr_offset = (PSRAM_VADDR_START - 0x3C000000) / MMU_PAGE_SIZE;

    /* Map 8MB of PSRAM (128 pages of 64KB each) */
    uint32_t num_pages = PSRAM_SIZE / MMU_PAGE_SIZE;

    for (uint32_t i = 0; i < num_pages; i++) {
        uint32_t entry = i               /* Physical page in PSRAM */
                       | MMU_ACCESS_SPIRAM  /* Access PSRAM, not flash */
                       | MMU_VALID;         /* Entry is valid */
        mmu_table[vaddr_offset + i] = entry;
    }

    /* Flush cache to apply MMU changes */
    REG_SET_BIT(EXTMEM_DCACHE_CTRL_REG, (1 << 2));  /* DCACHE_INVALIDATE */
    while (REG_READ(EXTMEM_DCACHE_CTRL_REG) & (1 << 3));  /* Wait for completion */

    /* Re-enable D-cache */
    REG_SET_BIT(EXTMEM_DCACHE_CTRL_REG, (1 << 0));

    /* Enable cache access to PSRAM region */
    REG_SET_BIT(SPI_MEM_CACHE_FCTRL_REG(PSRAM_SPI_NUM), (1 << 1));
}

/*
 * Verify PSRAM is working by writing and reading a test pattern.
 * Returns 0 on success, -1 on failure.
 */
int psram_test(void) {
    volatile uint32_t *psram = (volatile uint32_t *)PSRAM_VADDR_START;

    /* Write test pattern */
    psram[0] = 0xDEADBEEF;
    psram[1] = 0xCAFEBABE;
    psram[2] = 0x12345678;

    /* Memory barrier */
    __asm__ __volatile__("memw");

    /* Read back and verify */
    if (psram[0] != 0xDEADBEEF) return -1;
    if (psram[1] != 0xCAFEBABE) return -1;
    if (psram[2] != 0x12345678) return -1;

    /* Clear test data */
    psram[0] = 0;
    psram[1] = 0;
    psram[2] = 0;

    return 0;
}
