package module

// RegisterBuiltins registers all built-in modules with the resolver.
// Built-in modules are registered as source strings that get compiled
// when first imported.
func RegisterBuiltins(r *Resolver) {
	// core - primitives and basic objects (mostly handled by VM/eval builtins)
	// Empty for now since Integer, String, etc. are native
	r.RegisterBuiltin("core", coreSource)

	// io - I/O objects (Console and Transcript are already in eval builtins)
	r.RegisterBuiltin("io", ioSource)

	// collections - collection classes (stub for now)
	r.RegisterBuiltin("collections", collectionsSource)

	// task/tasksupervisor/timestamp/sdcard/wifi/led are runtime-backed globals.
	// These modules exist so `import` succeeds and documents module boundaries.
	r.RegisterBuiltin("task", taskSource)
	r.RegisterBuiltin("tasksupervisor", taskSupervisorSource)
	r.RegisterBuiltin("timestamp", timestampSource)
	r.RegisterBuiltin("sdcard", sdcardSource)
	r.RegisterBuiltin("wifi", wifiSource)
	r.RegisterBuiltin("led", ledSource)

	// hardware modules (stubs for now; full runtime objects in later phases)
	r.RegisterBuiltin("gpio", gpioSource)
	r.RegisterBuiltin("uart", uartSource)
	r.RegisterBuiltin("i2c", i2cSource)
	r.RegisterBuiltin("spi", spiSource)
}

// coreSource contains the picoceci source for the core module.
// Most primitives (Integer, String, Boolean, etc.) are native to the VM,
// so this module mainly provides convenience extensions.
const coreSource = `
" Core module - basic extensions to native types.
  Most primitives are implemented natively in the VM.
"
`

// ioSource contains the picoceci source for the io module.
// Console and Transcript are provided by eval/builtins.go as native objects.
const ioSource = `
" I/O module - input/output objects.
  Console and Transcript are provided by the runtime.
"

" ReadStream declaration (runtime behavior to be expanded in Phase 7+) "
object ReadStream {
}

" WriteStream declaration (runtime behavior to be expanded in Phase 7+) "
object WriteStream {
}
`

// collectionsSource contains the picoceci source for the collections module.
// This is a stub for Phase 4; full implementation in Phase 7.
const collectionsSource = `
" Collections module - collection classes.
  OrderedCollection, Dictionary, Set, Bag.
  Stub implementation for Phase 4.
"

" OrderedCollection - a growable array-like collection "
object OrderedCollection {
	let items: Array.

	initialize [
		items := #().
	]

	add: anObject [
		" Add an object to the end of the collection "
		items := items copyWith: anObject.
		^ anObject
	]

	size [
		^ items size
	]

	at: index [
		^ items at: index
	]

	do: aBlock [
		items do: aBlock.
	]
}

" Dictionary declaration "
object Dictionary {
}

" Set declaration "
object Set {
}

" Bag declaration "
object Bag {
}
`

const taskSource = `
" Task module - runtime-backed Task/Queue/Channel objects are globals. "
`

const taskSupervisorSource = `
" TaskSupervisor module - runtime-backed TaskSupervisor object is a global. "
`

const timestampSource = `
" Timestamp module - runtime-backed Timestamp/Duration objects are globals. "
`

const sdcardSource = `
" SDCard module - runtime-backed SDCard/File/Directory/Path objects are globals. "
`

const wifiSource = `
" WiFi module - runtime-backed WiFi singleton object is a global. "
`

const ledSource = `
" LED module - runtime-backed LED singleton object is a global. "
`

const gpioSource = `
" GPIO module stub - hardware-backed implementation planned for embedded runtime. "
object GPIO {
}
`

const uartSource = `
" UART module stub - hardware-backed implementation planned for embedded runtime. "
object UART {
}
`

const i2cSource = `
" I2C module stub - hardware-backed implementation planned for embedded runtime. "
object I2C {
}
`

const spiSource = `
" SPI module stub - hardware-backed implementation planned for embedded runtime. "
object SPI {
}
`
