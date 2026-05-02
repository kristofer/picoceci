/* Linker script for ESP32-S3 with PSRAM heap (N16R8: 16MB flash, 8MB PSRAM)
 *
 * This extends the standard esp32s3.ld to place the heap in PSRAM.
 *
 * Memory Map:
 *   Internal SRAM1 (416KB): stack, BSS, data, IRAM code
 *   PSRAM (8MB): heap for GC allocations
 *
 * IMPORTANT: This requires PSRAM to be initialized before the Go runtime
 * starts. The psram_init.S file contains the initialization code that must
 * run in the boot sequence.
 *
 * ESP32-S3 PSRAM addressing:
 *   - Octal PSRAM is accessed via cache at 0x3C000000-0x3DFFFFFF (data bus)
 *   - The MMU maps PSRAM pages after flash pages
 *   - With 16MB flash mapped at 0x3C000000, PSRAM starts at 0x3D000000
 */

MEMORY
{
    /* Internal SRAM1 - for stack, BSS, data (416KB) */
    DRAM (rw) : ORIGIN = 0x3FC88000, LENGTH = 416K
    IRAM (x)  : ORIGIN = 0x40378000, LENGTH = 416K

    /* Flash memory-mapped regions */
    DROM (r)  : ORIGIN = 0x3C000000, LENGTH = 16M  /* Flash data bus */
    IROM (rx) : ORIGIN = 0x42000000, LENGTH = 16M  /* Flash instruction bus */

    /* PSRAM - mapped after flash data region
     * 8MB octal PSRAM at 0x3D000000 (after 16MB flash at 0x3C000000)
     * This address assumes proper MMU configuration by psram_init code.
     */
    PSRAM (rw) : ORIGIN = 0x3D000000, LENGTH = 8M
}

ENTRY(call_start_cpu0)

SECTIONS
{
    /* Dummy section for flash image header alignment */
    .rodata_dummy (NOLOAD): ALIGN(4)
    {
        . += 0x18; /* esp_image_header_t */
        . += 0x8;  /* DROM segment header */
    } > DROM

    /* Constant global variables in flash */
    .rodata : ALIGN(4)
    {
        *(.rodata*)
        . = ALIGN (4);
        _drom_end = .;
    } >DROM

    /* Stack at bottom of DRAM (crash on overflow, not corrupt memory) */
    .stack (NOLOAD) :
    {
        . = ALIGN(16);
        . += _stack_size;
        _stack_top = .;
    } >DRAM

    /* Zero-initialized global variables */
    .bss (NOLOAD) : ALIGN(4)
    {
        . = ALIGN (4);
        _sbss = ABSOLUTE(.);
        *(.bss .bss.*)
        . = ALIGN (4);
        _ebss = ABSOLUTE(.);
    } >DRAM

    /* Initialized global variables (copied from flash by bootloader) */
    .data : ALIGN(4)
    {
        . = ALIGN (4);
        _sdata = ABSOLUTE(.);
        *(.data .data.*)
        *(.dram*)
        . = ALIGN (4);
        _edata = ABSOLUTE(.);
    } >DRAM

    /* Skip corresponding space in IRAM (dual-mapped with DRAM) */
    .iram_dummy (NOLOAD): ALIGN(4)
    {
        . += SIZEOF(.stack);
        . += SIZEOF(.bss);
        . += SIZEOF(.data);
    } > IRAM

    /* IRAM: boot code, interrupt vectors, time-critical code */
    .iram : ALIGN(4)
    {
        /* Boot entry point */
        *(.literal.call_start_cpu0)
        *(.text.call_start_cpu0)

        /* PSRAM initialization code - must run early */
        *(.literal.psram_init)
        *(.text.psram_init)

        /* Exception/interrupt vectors (0x400 aligned) */
        . = ALIGN(0x400);
        *(.text.exception_vectors)

        /* Level-1 interrupt handler */
        *(.literal._handle_level1)
        *(.text._handle_level1)

        /* WiFi/BLE IRAM sections */
        *(.iram*)
        *(.wifislprxiram*)
        *(.wifiextrairam*)
        *(.wifi0iram*)
        *(.wifislpiram*)
        *(.wifirxiram*)
        *(.wifiorslpiram*)
        *(.iram1*)
        *(.coexiram*)

        /* Init code (reclaimable) */
        __init_start = .;
        *(.init)
        __init_end = .;
        . = ALIGN(4);
        _iram_end = .;
    } >IRAM

    /* Flash offset alignment for IROM */
    .text_dummy (NOLOAD): ALIGN(4)
    {
        . += 0x18;
        . += SIZEOF(.rodata) + ((SIZEOF(.rodata) != 0) ? 0x8 : 0);
        . += SIZEOF(.data)   + ((SIZEOF(.data)   != 0) ? 0x8 : 0);
        . += SIZEOF(.iram)   + 0x8;
        . += 0x8;
    } > IROM

    /* Main code executed from flash */
    .text : ALIGN(4)
    {
        *(.literal .text)
        *(.literal.* .text.*)
        _irom_end = .;
    } >IROM

    /DISCARD/ :
    {
        *(.eh_frame)
    }
}

/* GC globals tracking - internal SRAM */
_globals_start = _sbss;
_globals_end = _edata;

/* HEAP IN PSRAM - This is the key change from standard linker script
 *
 * Place heap in PSRAM instead of internal SRAM:
 * - _heap_start: beginning of PSRAM region
 * - _heap_end: end of PSRAM region (8MB)
 *
 * This gives ~8MB for heap vs ~250KB in internal SRAM.
 */
_heap_start = ORIGIN(PSRAM);
_heap_end = ORIGIN(PSRAM) + LENGTH(PSRAM);

/* Keep some internal SRAM available for stack growth */
_stack_size = 8K;

/* ROM functions (same as standard esp32s3.ld) */
memset = 0x400011e8;
memcpy = 0x400011f4;
memmove = 0x40001200;
memcmp = 0x4000120c;

/* libgcc ROM functions */
__absvdi2 = 0x4000216c;
__absvsi2 = 0x40002178;
__adddf3 = 0x40002184;
__addsf3 = 0x40002190;
__addvdi3 = 0x4000219c;
__addvsi3 = 0x400021a8;
__ashldi3 = 0x400021b4;
__ashrdi3 = 0x400021c0;
__bswapdi2 = 0x400021cc;
__bswapsi2 = 0x400021d8;
__clear_cache = 0x400021e4;
__clrsbdi2 = 0x400021f0;
__clrsbsi2 = 0x400021fc;
__clzdi2 = 0x40002208;
__clzsi2 = 0x40002214;
__cmpdi2 = 0x40002220;
__ctzdi2 = 0x4000222c;
__ctzsi2 = 0x40002238;
__divdc3 = 0x40002244;
__divdf3 = 0x40002250;
__divdi3 = 0x4000225c;
__divsc3 = 0x40002268;
__divsf3 = 0x40002274;
__divsi3 = 0x40002280;
__eqdf2 = 0x4000228c;
__eqsf2 = 0x40002298;
__extendsfdf2 = 0x400022a4;
__ffsdi2 = 0x400022b0;
__ffssi2 = 0x400022bc;
__fixdfdi = 0x400022c8;
__fixdfsi = 0x400022d4;
__fixsfdi = 0x400022e0;
__fixsfsi = 0x400022ec;
__fixunsdfsi = 0x400022f8;
__fixunssfdi = 0x40002304;
__fixunssfsi = 0x40002310;
__floatdidf = 0x4000231c;
__floatdisf = 0x40002328;
__floatsidf = 0x40002334;
__floatsisf = 0x40002340;
__floatundidf = 0x4000234c;
__floatundisf = 0x40002358;
__floatunsidf = 0x40002364;
__floatunsisf = 0x40002370;
__gcc_bcmp = 0x4000237c;
__gedf2 = 0x40002388;
__gesf2 = 0x40002394;
__gtdf2 = 0x400023a0;
__gtsf2 = 0x400023ac;
__ledf2 = 0x400023b8;
__lesf2 = 0x400023c4;
__lshrdi3 = 0x400023d0;
__ltdf2 = 0x400023dc;
__ltsf2 = 0x400023e8;
__moddi3 = 0x400023f4;
__modsi3 = 0x40002400;
__muldc3 = 0x4000240c;
__muldf3 = 0x40002418;
__muldi3 = 0x40002424;
__mulsc3 = 0x40002430;
__mulsf3 = 0x4000243c;
__mulsi3 = 0x40002448;
__mulvdi3 = 0x40002454;
__mulvsi3 = 0x40002460;
__nedf2 = 0x4000246c;
__negdf2 = 0x40002478;
__negdi2 = 0x40002484;
__negsf2 = 0x40002490;
__negvdi2 = 0x4000249c;
__negvsi2 = 0x400024a8;
__nesf2 = 0x400024b4;
__paritysi2 = 0x400024c0;
__popcountdi2 = 0x400024cc;
__popcountsi2 = 0x400024d8;
__powidf2 = 0x400024e4;
__powisf2 = 0x400024f0;
__subdf3 = 0x400024fc;
__subsf3 = 0x40002508;
__subvdi3 = 0x40002514;
__subvsi3 = 0x40002520;
__truncdfsf2 = 0x4000252c;
__ucmpdi2 = 0x40002538;
__udivdi3 = 0x40002544;
__udivmoddi4 = 0x40002550;
__udivsi3 = 0x4000255c;
__udiv_w_sdiv = 0x40002568;
__umoddi3 = 0x40002574;
__umodsi3 = 0x40002580;
__unorddf2 = 0x4000258c;
__unordsf2 = 0x40002598;

/* Interrupt handling */
intr_matrix_set = 0x40001b54;
ets_intr_lock = 0x40001b60;
ets_intr_unlock = 0x40001b6c;
ets_isr_attach = 0x40001b78;
ets_isr_unmask = 0x40001b90;
