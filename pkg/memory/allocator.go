// Package memory implements the picoceci object allocator.
//
// Default strategy: reference counting with a simple cycle collector.
// On TinyGo/MCU targets the allocator uses a fixed-size heap arena
// drawn from PSRAM.
package memory

import (
	"sync/atomic"

	"github.com/kristofer/picoceci/pkg/object"
)

// Retain increments the reference count of an object.
// It is safe to call on nil or singleton objects (nil, true, false).
func Retain(o *object.Object) {
	if o == nil || isSingleton(o) {
		return
	}
	atomic.AddInt32(&o.RefCount, 1)
}

// Release decrements the reference count of an object.
// If the count reaches zero, the object's resources are released.
// It is safe to call on nil or singleton objects.
func Release(o *object.Object) {
	if o == nil || isSingleton(o) {
		return
	}

	newCount := atomic.AddInt32(&o.RefCount, -1)
	if newCount <= 0 {
		releaseResources(o)
	}
}

// RefCount returns the current reference count of an object.
// Returns 0 for nil or singleton objects.
func RefCount(o *object.Object) int32 {
	if o == nil || isSingleton(o) {
		return 0
	}
	return atomic.LoadInt32(&o.RefCount)
}

// Alloc creates a new reference-counted object of the given kind.
// The initial reference count is 1.
// For singletons (nil, bool), returns the existing singleton.
func Alloc(kind object.Kind) *object.Object {
	// For singletons, return the existing singleton
	switch kind {
	case object.KindNil:
		return object.Nil
	case object.KindBool:
		return object.False // caller should use object.BoolObject instead
	}

	o := &object.Object{
		Kind:     kind,
		RefCount: 1,
	}

	// Initialize container fields as needed
	switch kind {
	case object.KindArray:
		o.Items = make([]*object.Object, 0)
	case object.KindByteArray:
		o.Bytes = make([]byte, 0)
	case object.KindObject:
		o.Slots = make(map[string]*object.Object)
		o.Methods = make(map[string]*object.MethodDef)
	case object.KindBlock:
		o.Params = make([]string, 0)
		o.Locals = make([]string, 0)
	}

	return o
}

// AllocInt creates an integer object with refcount=1.
func AllocInt(v int64) *object.Object {
	return &object.Object{
		Kind:     object.KindSmallInt,
		IVal:     v,
		RefCount: 1,
	}
}

// AllocFloat creates a float object with refcount=1.
func AllocFloat(v float64) *object.Object {
	return &object.Object{
		Kind:     object.KindFloat,
		FVal:     v,
		RefCount: 1,
	}
}

// AllocString creates a string object with refcount=1.
func AllocString(s string) *object.Object {
	return &object.Object{
		Kind:     object.KindString,
		SVal:     s,
		RefCount: 1,
	}
}

// AllocSymbol creates a symbol object with refcount=1.
// In a full implementation, symbols should be interned.
func AllocSymbol(s string) *object.Object {
	return &object.Object{
		Kind:     object.KindSymbol,
		SVal:     s,
		RefCount: 1,
	}
}

// AllocChar creates a character object with refcount=1.
func AllocChar(r rune) *object.Object {
	return &object.Object{
		Kind:     object.KindChar,
		RVal:     r,
		RefCount: 1,
	}
}

// AllocArray creates an array object of the given size with refcount=1.
// All elements are initialized to nil.
func AllocArray(size int) *object.Object {
	items := make([]*object.Object, size)
	for i := range items {
		items[i] = object.Nil
	}
	return &object.Object{
		Kind:     object.KindArray,
		Items:    items,
		RefCount: 1,
	}
}

// AllocByteArray creates a byte array object with refcount=1.
func AllocByteArray(data []byte) *object.Object {
	cp := make([]byte, len(data))
	copy(cp, data)
	return &object.Object{
		Kind:     object.KindByteArray,
		Bytes:    cp,
		RefCount: 1,
	}
}

// isSingleton returns true if the object is a singleton (nil, true, false).
func isSingleton(o *object.Object) bool {
	return o == object.Nil || o == object.True || o == object.False
}

// releaseResources releases all resources held by an object.
// This is called when the reference count reaches zero.
func releaseResources(o *object.Object) {
	switch o.Kind {
	case object.KindArray:
		// Release all array elements
		for i, item := range o.Items {
			Release(item)
			o.Items[i] = nil
		}
		o.Items = nil

	case object.KindObject:
		// Release all slot values
		for k, v := range o.Slots {
			Release(v)
			delete(o.Slots, k)
		}
		o.Slots = nil
		o.Methods = nil
		o.ComposedMethods = nil

	case object.KindBlock:
		// Release captured environment if present
		// Note: The actual Env release would need to be handled
		// by the eval package since Env is an interface{} here
		o.Env = nil
		o.Body = nil
		o.Params = nil
		o.Locals = nil

	case object.KindByteArray:
		o.Bytes = nil

	case object.KindString, object.KindSymbol:
		// Strings are immutable and managed by Go's GC
		o.SVal = ""
	}

	// Clear primitive values
	o.IVal = 0
	o.FVal = 0
	o.BVal = false
	o.RVal = 0
}
