package widgets

import (
	"testing"

	ui "github.com/gizak/termui"
)

func TestInputOwnerRendersChanges(t *testing.T) {
	i := NewInput()
	var changes []string
	i.OnChange = func(s string) {
		if i.Data != s {
			t.Error("callback ran before input was updated")
		}
		changes = append(changes, s)
	}
	// No terminal or stream is initialized: the callback must own rendering.
	for _, key := range []string{"a", "b", "C-8", "C-8", "C-8"} {
		i.KeyPress(ui.Event{Path: "/sys/kbd/" + key})
	}
	want := []string{"a", "ab", "a", ""}
	if len(changes) != len(want) {
		t.Fatalf("changes = %q, want %q", changes, want)
	}
	for n := range want {
		if changes[n] != want[n] {
			t.Fatalf("changes = %q, want %q", changes, want)
		}
	}
}
