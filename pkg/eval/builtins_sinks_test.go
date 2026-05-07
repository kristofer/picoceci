package eval

import (
	"bytes"
	"testing"

	"github.com/kristofer/picoceci/pkg/object"
)

func callNoArgNative(t *testing.T, recv *object.Object, selector string) {
	t.Helper()
	m, ok := recv.Methods[selector]
	if !ok || m == nil || m.Native == nil {
		t.Fatalf("missing native method %q", selector)
	}
	if _, err := m.Native(recv, nil); err != nil {
		t.Fatalf("call %q failed: %v", selector, err)
	}
}

func callOneArgNative(t *testing.T, recv *object.Object, selector string, arg *object.Object) {
	t.Helper()
	m, ok := recv.Methods[selector]
	if !ok || m == nil || m.Native == nil {
		t.Fatalf("missing native method %q", selector)
	}
	if _, err := m.Native(recv, []*object.Object{arg}); err != nil {
		t.Fatalf("call %q failed: %v", selector, err)
	}
}

func TestInitialGlobalsWithSinks_SplitConsoleAndTranscript(t *testing.T) {
	var consoleBuf bytes.Buffer
	var transcriptBuf bytes.Buffer

	globals := InitialGlobalsWithSinks(GlobalSinks{
		ConsoleWriter:    &consoleBuf,
		TranscriptWriter: &transcriptBuf,
	})

	console := globals["Console"]
	transcript := globals["Transcript"]
	if console == nil || transcript == nil {
		t.Fatal("expected Console and Transcript globals")
	}

	callOneArgNative(t, console, "println:", object.StringObject("console"))
	callNoArgNative(t, console, "nl")
	callOneArgNative(t, transcript, "println:", object.StringObject("transcript"))

	if got := consoleBuf.String(); got != "console\n\n" {
		t.Fatalf("console sink output = %q, want %q", got, "console\\n\\n")
	}
	if got := transcriptBuf.String(); got != "transcript\n" {
		t.Fatalf("transcript sink output = %q, want %q", got, "transcript\\n")
	}
}

func TestInitialGlobalsWithSinks_TranscriptFallsBackToConsoleSink(t *testing.T) {
	var consoleBuf bytes.Buffer

	globals := InitialGlobalsWithSinks(GlobalSinks{ConsoleWriter: &consoleBuf})
	transcript := globals["Transcript"]
	if transcript == nil {
		t.Fatal("expected Transcript global")
	}

	callOneArgNative(t, transcript, "print:", object.StringObject("hello"))

	if got := consoleBuf.String(); got != "hello" {
		t.Fatalf("fallback sink output = %q, want %q", got, "hello")
	}
}
