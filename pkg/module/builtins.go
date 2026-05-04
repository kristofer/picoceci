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
	| items: Array |

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
`
