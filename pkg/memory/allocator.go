// Package memory implements the picoceci object allocator.
//
// Default strategy: reference counting with a simple cycle collector.
// On TinyGo/MCU targets the allocator uses a fixed-size heap arena
// drawn from PSRAM.
//
// Phase 3 deliverable — see IMPLEMENTATION_PLAN.md.
package memory

import "github.com/kristofer/picoceci/pkg/object"

// Retain increments the reference count of o.
// It is safe to call with a nil pointer.
func Retain(o *object.Object) {
	if o == nil {
		return
	}
	o.RefCount++
}

// Release decrements the reference count of o.
// When the count reaches zero all referenced objects are recursively released.
// It is safe to call with a nil pointer.
func Release(o *object.Object) {
	if o == nil {
		return
	}
	o.RefCount--
	if o.RefCount > 0 {
		return
	}
	// Release all held references so the GC can reclaim them.
	for _, v := range o.Slots {
		Release(v)
	}
	for _, v := range o.Items {
		Release(v)
	}
}

// Alloc creates a new reference-counted object of the given kind.
// The initial reference count is 1.
func Alloc(kind object.Kind) *object.Object {
	o := &object.Object{Kind: kind, RefCount: 1}
	switch kind {
	case object.KindObject:
		o.Slots = make(map[string]*object.Object)
		o.Methods = make(map[string]*object.MethodDef)
	case object.KindArray:
		o.Items = make([]*object.Object, 0)
	}
	return o
}
