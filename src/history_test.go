package fzf

import (
	"fmt"
	"os"
	"runtime"
	"testing"
)

func TestHistory(t *testing.T) {
	maxHistory := 50

	// Invalid arguments
	var paths []string
	if runtime.GOOS == "windows" {
		// GOPATH should exist, so we shouldn't be able to override it
		paths = []string{os.Getenv("GOPATH")}
	} else {
		paths = []string{"/etc", "/proc"}
	}

	for _, path := range paths {
		if _, e := NewHistory(path, maxHistory); e == nil {
			t.Error("Error expected for: " + path)
		}
	}

	f, _ := os.CreateTemp("", "fzf-history")
	f.Close()

	{ // Append lines
		h, _ := NewHistory(f.Name(), maxHistory)
		for i := 0; i < maxHistory+10; i++ {
			h.append("foobar")
		}
	}
	{ // Read lines
		h, _ := NewHistory(f.Name(), maxHistory)
		if len(h.lines) != maxHistory+1 {
			t.Errorf("Expected: %d, actual: %d\n", maxHistory+1, len(h.lines))
		}
		for i := range maxHistory {
			if h.lines[i] != "foobar" {
				t.Error("Expected: foobar, actual: " + h.lines[i])
			}
		}
	}
	{ // Append lines
		h, _ := NewHistory(f.Name(), maxHistory)
		h.append("barfoo")
		h.append("")
		h.append("foobarbaz")
	}
	{ // Read lines again
		h, _ := NewHistory(f.Name(), maxHistory)
		if len(h.lines) != maxHistory+1 {
			t.Errorf("Expected: %d, actual: %d\n", maxHistory+1, len(h.lines))
		}
		compare := func(idx int, exp string) {
			if h.lines[idx] != exp {
				t.Errorf("Expected: %s, actual: %s\n", exp, h.lines[idx])
			}
		}
		compare(maxHistory-3, "foobar")
		compare(maxHistory-2, "barfoo")
		compare(maxHistory-1, "foobarbaz")
	}
}

func TestHistoryTruncateOnLoad(t *testing.T) {
	f, _ := os.CreateTemp("", "fzf-history")
	f.Close()

	// Write more lines than maxSize to file
	maxSize := 5
	{ // Create history with large maxSize first
		h, _ := NewHistory(f.Name(), 100)
		for i := 0; i < 10; i++ {
			h.append(fmt.Sprintf("line%d", i))
		}
	}
	{ // Load with smaller maxSize, should truncate to most recent entries
		h, _ := NewHistory(f.Name(), maxSize)
		if len(h.lines) != maxSize+1 {
			t.Errorf("Expected %d lines, got %d", maxSize+1, len(h.lines))
		}
		// Should keep the most recent entries (line5..line9)
		for i := 0; i < maxSize; i++ {
			expected := fmt.Sprintf("line%d", i+5)
			if h.lines[i] != expected {
				t.Errorf("lines[%d]: expected %s, got %s", i, expected, h.lines[i])
			}
		}
		// Cursor should point to the empty trailing entry
		if h.cursor != maxSize {
			t.Errorf("cursor: expected %d, got %d", maxSize, h.cursor)
		}
	}
}

func TestHistoryNavigationAfterTruncation(t *testing.T) {
	f, _ := os.CreateTemp("", "fzf-history")
	f.Close()

	maxSize := 3
	{ // Write entries
		h, _ := NewHistory(f.Name(), 100)
		for i := 1; i <= 5; i++ {
			h.append(fmt.Sprintf("entry%d", i))
		}
	}
	{ // Load with smaller maxSize and navigate
		h, _ := NewHistory(f.Name(), maxSize)

		// Should start at end (empty current input)
		if cur := h.current(); cur != "" {
			t.Errorf("current() at start: expected empty, got %q", cur)
		}

		// Navigate backwards through all entries without blanks or errors
		prev := h.previous()
		if prev != "entry5" {
			t.Errorf("previous() 1: expected entry5, got %q", prev)
		}
		prev = h.previous()
		if prev != "entry4" {
			t.Errorf("previous() 2: expected entry4, got %q", prev)
		}
		prev = h.previous()
		if prev != "entry3" {
			t.Errorf("previous() 3: expected entry3, got %q", prev)
		}
		// At the beginning, should not go further
		prev = h.previous()
		if prev != "entry3" {
			t.Errorf("previous() at boundary: expected entry3, got %q", prev)
		}

		// Navigate forward back to current
		next := h.next()
		if next != "entry4" {
			t.Errorf("next() 1: expected entry4, got %q", next)
		}
		next = h.next()
		if next != "entry5" {
			t.Errorf("next() 2: expected entry5, got %q", next)
		}
		next = h.next()
		if next != "" {
			t.Errorf("next() to end: expected empty, got %q", next)
		}
	}
}

func TestHistoryCursorResetAfterAppend(t *testing.T) {
	f, _ := os.CreateTemp("", "fzf-history")
	f.Close()

	maxSize := 3
	h, _ := NewHistory(f.Name(), maxSize)
	h.append("a")
	h.append("b")
	h.append("c")

	// Navigate backward and override
	h.previous() // cursor=2, "c"
	h.override("modified_c")
	h.previous() // cursor=1, "b"

	// Append should reset cursor and modified
	h.append("d")

	if h.cursor != len(h.lines)-1 {
		t.Errorf("cursor after append: expected %d, got %d", len(h.lines)-1, h.cursor)
	}
	// modified should be cleared, so current() returns the raw line
	if cur := h.current(); cur != "" {
		t.Errorf("current() after append: expected empty, got %q", cur)
	}
	// Navigate should work correctly after append+truncation
	prev := h.previous()
	if prev != "d" {
		t.Errorf("previous() after append: expected d, got %q", prev)
	}
	prev = h.previous()
	if prev != "c" {
		t.Errorf("previous() after append 2: expected c, got %q", prev)
	}
}
