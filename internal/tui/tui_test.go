package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mrcawood/History_eXtended/internal/search"
)

func keyMsg(s string) tea.KeyMsg {
	switch s {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
}

func TestNextFilterModeInSearch(t *testing.T) {
	if search.NextFilter(search.FilterGlobal) != search.FilterHost {
		t.Fatal("filter cycle")
	}
	if search.NextMode(search.ModeFuzzy) != search.ModePrefix {
		t.Fatal("mode cycle")
	}
}

func TestEnterAcceptIntent(t *testing.T) {
	row := search.Row{Cmd: "make build"}

	// Default (enter_accept false): Enter inserts for edit, Tab runs.
	m := model{enterAccept: false, rows: []search.Row{row}}
	got, _ := m.handleKey(keyMsg("enter"))
	gm := got.(model)
	if gm.accepted != "make build" || gm.runRequested {
		t.Fatalf("enter (default): accepted=%q run=%v, want edit", gm.accepted, gm.runRequested)
	}

	m = model{enterAccept: false, rows: []search.Row{row}}
	got, _ = m.handleKey(keyMsg("tab"))
	gm = got.(model)
	if !gm.runRequested {
		t.Fatal("tab (default): expected run intent")
	}

	// enter_accept true: Enter runs, Tab edits.
	m = model{enterAccept: true, rows: []search.Row{row}}
	got, _ = m.handleKey(keyMsg("enter"))
	gm = got.(model)
	if !gm.runRequested {
		t.Fatal("enter (enter_accept): expected run intent")
	}

	m = model{enterAccept: true, rows: []search.Row{row}}
	got, _ = m.handleKey(keyMsg("tab"))
	gm = got.(model)
	if gm.runRequested {
		t.Fatal("tab (enter_accept): expected edit intent")
	}
}

func TestFormatRowExitColor(t *testing.T) {
	exit := 1
	m := model{width: 100, rows: []search.Row{{
		Cmd: "false", ExitCode: &exit, StartedAt: 0,
	}}}
	line := m.formatRow(m.rows[0], 80)
	if line == "" {
		t.Fatal("empty line")
	}
}
