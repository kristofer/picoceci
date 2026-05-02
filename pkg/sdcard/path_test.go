package sdcard_test

import (
	"testing"

	"github.com/kristofer/picoceci/pkg/sdcard"
)

func TestPathBasename(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"/sdcard/data/log.csv", "log.csv"},
		{"/sdcard/data/", "data"},
		{"/sdcard/file", "file"},
		{"file.txt", "file.txt"},
		{"/", "/"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			p := sdcard.PathFrom(tt.path)
			if got := p.Basename(); got != tt.expected {
				t.Errorf("Basename() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestPathDirname(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"/sdcard/data/log.csv", "/sdcard/data"},
		{"/sdcard/data/", "/sdcard/data"},
		{"/sdcard/file", "/sdcard"},
		{"file.txt", "."},
		{"/", "/"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			p := sdcard.PathFrom(tt.path)
			if got := p.Dirname().String(); got != tt.expected {
				t.Errorf("Dirname() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestPathExtension(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"/sdcard/data/log.csv", "csv"},
		{"/sdcard/data/log.tar.gz", "gz"},
		{"/sdcard/noext", ""},
		{"file.TXT", "TXT"},
		{".hidden", "hidden"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			p := sdcard.PathFrom(tt.path)
			if got := p.Extension(); got != tt.expected {
				t.Errorf("Extension() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestPathStem(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"/sdcard/data/log.csv", "log"},
		{"/sdcard/data/log.tar.gz", "log.tar"},
		{"/sdcard/noext", "noext"},
		{"file.txt", "file"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			p := sdcard.PathFrom(tt.path)
			if got := p.Stem(); got != tt.expected {
				t.Errorf("Stem() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestPathJoin(t *testing.T) {
	tests := []struct {
		base     string
		child    string
		expected string
	}{
		{"/sdcard/data", "log.csv", "/sdcard/data/log.csv"},
		{"/sdcard", "data", "/sdcard/data"},
		{"/sdcard/", "file", "/sdcard/file"},
	}

	for _, tt := range tests {
		t.Run(tt.base+"+"+tt.child, func(t *testing.T) {
			p := sdcard.PathFrom(tt.base)
			if got := p.Join(tt.child).String(); got != tt.expected {
				t.Errorf("Join() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestPathWithSuffix(t *testing.T) {
	p := sdcard.PathFrom("/sdcard/data/log")
	got := p.WithSuffix(".csv").String()
	expected := "/sdcard/data/log.csv"
	if got != expected {
		t.Errorf("WithSuffix() = %q, want %q", got, expected)
	}
}

func TestPathIsAbsolute(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"/sdcard/data", true},
		{"relative/path", false},
		{"file.txt", false},
		{"/", true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			p := sdcard.PathFrom(tt.path)
			if got := p.IsAbsolute(); got != tt.expected {
				t.Errorf("IsAbsolute() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestPathClean(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"/sdcard/data/../data/log.csv", "/sdcard/data/log.csv"},
		{"/sdcard/./data", "/sdcard/data"},
		{"/sdcard//data", "/sdcard/data"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			p := sdcard.PathFrom(tt.path)
			if got := p.Clean().String(); got != tt.expected {
				t.Errorf("Clean() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestPathSegments(t *testing.T) {
	tests := []struct {
		path     string
		expected []string
	}{
		{"/sdcard/data/log.csv", []string{"sdcard", "data", "log.csv"}},
		{"/sdcard", []string{"sdcard"}},
		{"/", nil},
		{"relative/path", []string{"relative", "path"}},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			p := sdcard.PathFrom(tt.path)
			got := p.Segments()
			if len(got) != len(tt.expected) {
				t.Errorf("Segments() = %v, want %v", got, tt.expected)
				return
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("Segments()[%d] = %q, want %q", i, got[i], tt.expected[i])
				}
			}
		})
	}
}
