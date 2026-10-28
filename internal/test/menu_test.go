package test

import (
	"testing"

	"github.com/sinueh/crudding/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
)

func TestMenuNavigationSelectionAndConfirmation(t *testing.T) {
	current := tui.NewMenu("test", []string{"python", "go", "node"})

	updated, _ := current.Update(tea.KeyMsg{Type: tea.KeyDown})
	current = updated.(tui.Menu)
	if current.Cursor() != 1 {
		t.Fatalf("cursor = %d, want 1", current.Cursor())
	}
	updated, _ = current.Update(tea.KeyMsg{Type: tea.KeySpace})
	current = updated.(tui.Menu)
	updated, command := current.Update(tea.KeyMsg{Type: tea.KeyEnter})
	current = updated.(tui.Menu)
	if !current.Confirmed() || command == nil || current.Selected() != 1 {
		t.Fatal("menu should select and confirm the focused option")
	}
}
