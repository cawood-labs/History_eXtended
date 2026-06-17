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

	// TUI renders to stderr (stdout may be a pipe when invoked from zsh widgets).
	out := os.Stderr
	if !term.IsTerminal(int(out.Fd())) {
		if tty, err := os.OpenFile("/dev/tty", os.O_WRONLY, 0); err == nil {
			defer func() { _ = tty.Close() }()
			out = tty
		}
	}
	initStyles(out)

	m := newModel(opts)
	// Render to stderr so zsh widgets can capture the selected command on stdout
	// via $(hx search -i </dev/tty 2>/dev/tty) without swallowing TUI frames.
	progOpts := append(programOptions(m), tea.WithOutput(out))
	p := tea.NewProgram(m, progOpts...)
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
