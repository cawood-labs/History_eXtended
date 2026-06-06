package tui

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"
)

// Result is the outcome of an interactive search session.
type Result struct {
	Cmd       string
	Cancelled bool
	// RunRequested is true when the user chose to run the command immediately
	// (vs insert for editing). Determined by enter_accept config and Enter/Tab.
	RunRequested bool
}

// Run starts the interactive TUI. Selected command is in Result.Cmd when not cancelled.
func Run(opts Options) (Result, error) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return Result{}, ErrNotTTY
	}

	m := newModel(opts)
	p := tea.NewProgram(m, programOptions(m)...)
	final, err := p.Run()
	if err != nil {
		return Result{}, err
	}
	fm := final.(model)
	if fm.quitting && fm.accepted == "" {
		return Result{Cancelled: true}, nil
	}
	return Result{Cmd: fm.accepted, RunRequested: fm.runRequested}, nil
}

func programOptions(m model) []tea.ProgramOption {
	if m.inline {
		return nil
	}
	return []tea.ProgramOption{tea.WithAltScreen()}
}
