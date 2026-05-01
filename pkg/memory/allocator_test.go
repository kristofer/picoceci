package memory

import (
	"testing"

	"github.com/kristofer/picoceci/pkg/object"
)

func TestRetainRelease(t *testing.T) {
	o := AllocInt(42)

	if RefCount(o) != 1 {
		t.Errorf("expected initial refcount 1, got %d", RefCount(o))
	}

	Retain(o)
	if RefCount(o) != 2 {
		t.Errorf("expected refcount 2 after Retain, got %d", RefCount(o))
	}

	Release(o)
	if RefCount(o) != 1 {
		t.Errorf("expected refcount 1 after Release, got %d", RefCount(o))
	}

	Release(o)
	if RefCount(o) != 0 {
		t.Errorf("expected refcount 0 after final Release, got %d", RefCount(o))
	}
}

func TestRetainReleaseSingletons(t *testing.T) {
	// Singletons should not be affected by retain/release
	Retain(object.Nil)
	Retain(object.True)
	Retain(object.False)

	if RefCount(object.Nil) != 0 {
		t.Error("nil should have refcount 0")
	}
	if RefCount(object.True) != 0 {
		t.Error("true should have refcount 0")
	}
	if RefCount(object.False) != 0 {
		t.Error("false should have refcount 0")
	}

	Release(object.Nil)
	Release(object.True)
	Release(object.False)
	// Should not panic or cause issues
}

func TestRetainReleaseNil(t *testing.T) {
	// Should not panic
	Retain(nil)
	Release(nil)
	if RefCount(nil) != 0 {
		t.Error("nil pointer should have refcount 0")
	}
}

func TestAllocInt(t *testing.T) {
	o := AllocInt(42)
	if o.Kind != object.KindSmallInt {
		t.Errorf("expected KindSmallInt, got %v", o.Kind)
	}
	if o.IVal != 42 {
		t.Errorf("expected 42, got %d", o.IVal)
	}
	if RefCount(o) != 1 {
		t.Errorf("expected refcount 1, got %d", RefCount(o))
	}
}

func TestAllocFloat(t *testing.T) {
	o := AllocFloat(3.14)
	if o.Kind != object.KindFloat {
		t.Errorf("expected KindFloat, got %v", o.Kind)
	}
	if o.FVal != 3.14 {
		t.Errorf("expected 3.14, got %f", o.FVal)
	}
	if RefCount(o) != 1 {
		t.Errorf("expected refcount 1, got %d", RefCount(o))
	}
}

func TestAllocString(t *testing.T) {
	o := AllocString("hello")
	if o.Kind != object.KindString {
		t.Errorf("expected KindString, got %v", o.Kind)
	}
	if o.SVal != "hello" {
		t.Errorf("expected 'hello', got %q", o.SVal)
	}
	if RefCount(o) != 1 {
		t.Errorf("expected refcount 1, got %d", RefCount(o))
	}
}

func TestAllocSymbol(t *testing.T) {
	o := AllocSymbol("foo")
	if o.Kind != object.KindSymbol {
		t.Errorf("expected KindSymbol, got %v", o.Kind)
	}
	if o.SVal != "foo" {
		t.Errorf("expected 'foo', got %q", o.SVal)
	}
}

func TestAllocChar(t *testing.T) {
	o := AllocChar('A')
	if o.Kind != object.KindChar {
		t.Errorf("expected KindChar, got %v", o.Kind)
	}
	if o.RVal != 'A' {
		t.Errorf("expected 'A', got %c", o.RVal)
	}
}

func TestAllocArray(t *testing.T) {
	o := AllocArray(3)
	if o.Kind != object.KindArray {
		t.Errorf("expected KindArray, got %v", o.Kind)
	}
	if len(o.Items) != 3 {
		t.Errorf("expected 3 items, got %d", len(o.Items))
	}
	for i, item := range o.Items {
		if item != object.Nil {
			t.Errorf("item %d should be nil", i)
		}
	}
}

func TestAllocByteArray(t *testing.T) {
	data := []byte{1, 2, 3}
	o := AllocByteArray(data)
	if o.Kind != object.KindByteArray {
		t.Errorf("expected KindByteArray, got %v", o.Kind)
	}
	if len(o.Bytes) != 3 {
		t.Errorf("expected 3 bytes, got %d", len(o.Bytes))
	}
	// Verify it's a copy
	data[0] = 99
	if o.Bytes[0] != 1 {
		t.Error("byte array should be a copy, not a reference")
	}
}

func TestAllocSingletonKinds(t *testing.T) {
	nilObj := Alloc(object.KindNil)
	if nilObj != object.Nil {
		t.Error("Alloc(KindNil) should return object.Nil singleton")
	}

	boolObj := Alloc(object.KindBool)
	if boolObj != object.False {
		t.Error("Alloc(KindBool) should return object.False singleton")
	}
}

func TestReleaseArrayRecursive(t *testing.T) {
	// Create an array with nested objects
	arr := AllocArray(2)
	child1 := AllocInt(1)
	child2 := AllocInt(2)

	arr.Items[0] = child1
	arr.Items[1] = child2

	// Verify initial refcounts
	if RefCount(child1) != 1 {
		t.Errorf("child1 refcount should be 1, got %d", RefCount(child1))
	}

	// Release the array
	Release(arr)

	// After release, array items should be nil and children released
	if arr.Items != nil {
		t.Error("array items should be nil after release")
	}
}

func TestReleaseObjectSlots(t *testing.T) {
	obj := Alloc(object.KindObject)
	slot := AllocInt(42)
	obj.Slots["value"] = slot

	// Release the object
	Release(obj)

	// Slots should be cleared
	if obj.Slots != nil {
		t.Error("object slots should be nil after release")
	}
}

func TestIsSingleton(t *testing.T) {
	if !isSingleton(object.Nil) {
		t.Error("object.Nil should be a singleton")
	}
	if !isSingleton(object.True) {
		t.Error("object.True should be a singleton")
	}
	if !isSingleton(object.False) {
		t.Error("object.False should be a singleton")
	}

	regular := AllocInt(42)
	if isSingleton(regular) {
		t.Error("regular object should not be a singleton")
	}
}
