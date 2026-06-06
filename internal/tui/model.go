package tui

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mrcawood/History_eXtended/internal/config"
	"github.com/mrcawood/History_eXtended/internal/search"
)

// Options configures the interactive search TUI.
type Options struct {
	Conn *sql.DB
	Cfg  *config.Config
	Req  search.Request
}

type model struct {
	conn   *sql.DB
	cfg    *config.Config
	req    search.Request
	input  textinput.Model
	rows   []search.Row
	cursor int
	width  int
	height int

	preview      string
	searching    bool
	inspector    bool
	accepted     string
	runRequested bool
	enterAccept  bool
	quitting     bool

	inline       bool
	inlineHeight int
}

func newModel(opts Options) model {
	ti := textinput.New()
	ti.Placeholder = "search history…"
	ti.Prompt = "hx> "
	ti.CharLimit = 512
	ti.SetValue(opts.Req.Query)
	ti.Focus()

	inline := false
	inlineH := 15
	enterAccept := false
	if opts.Cfg != nil {
		if strings.EqualFold(opts.Cfg.Search.UIStyle, "inline") {
			inline = true
		}
		if opts.Cfg.Search.InlineHeight > 0 {
			inlineH = opts.Cfg.Search.InlineHeight
		}
		enterAccept = opts.Cfg.Search.EnterAccept
	}

	return model{
		conn:         opts.Conn,
		cfg:          opts.Cfg,
		req:          opts.Req,
		input:        ti,
		inline:       inline,
		inlineHeight: inlineH,
		enterAccept:  enterAccept,
		searching:    true,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.runSearch())
}

func (m model) runSearch() tea.Cmd {
	req := m.req
	req.Query = m.input.Value()
	conn := m.conn
	cfg := m.cfg
	return func() tea.Msg {
		rows, err := search.Search(context.Background(), conn, cfg, req)
		return searchDoneMsg{rows: rows, err: err}
	}
}

func (m model) loadPreview(eventID int64) tea.Cmd {
	if eventID <= 0 {
		return nil
	}
	conn := m.conn
	return func() tea.Msg {
		d, err := search.GetEvent(conn, eventID)
		if err != nil {
			return detailDoneMsg{err: err}
		}
		return detailDoneMsg{text: search.FormatDetail(d)}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.inspector {
			return m.handleInspectorKey(msg)
		}
		return m.handleKey(msg)
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case searchDoneMsg:
		m.searching = false
		if msg.err != nil {
			m.preview = "search error: " + msg.err.Error()
			return m, nil
		}
		m.rows = msg.rows
		if m.cursor >= len(m.rows) {
			m.cursor = max(0, len(m.rows)-1)
		}
		if len(m.rows) > 0 {
			return m, m.loadPreview(m.rows[m.cursor].EventID)
		}
		m.preview = "(no matches)"
		return m, nil
	case detailDoneMsg:
		if msg.err != nil {
			m.preview = msg.err.Error()
		} else {
			m.preview = msg.text
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m model) handleInspectorKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+o", "q":
		m.inspector = false
		return m, nil
	case "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	}
	return m, nil
}

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "esc":
		m.quitting = true
		return m, tea.Quit
	case "enter":
		if len(m.rows) > 0 && m.cursor < len(m.rows) {
			m.accepted = m.rows[m.cursor].Cmd
			m.runRequested = m.enterAccept
			return m, tea.Quit
		}
		return m, nil
	case "tab":
		if len(m.rows) > 0 && m.cursor < len(m.rows) {
			m.accepted = m.rows[m.cursor].Cmd
			m.runRequested = !m.enterAccept
			return m, tea.Quit
		}
		return m, nil
	case "up", "ctrl+p":
		m.cursor = max(0, m.cursor-1)
		return m, m.loadPreview(m.currentEventID())
	case "down", "ctrl+n":
		m.cursor = min(len(m.rows)-1, m.cursor+1)
		return m, m.loadPreview(m.currentEventID())
	case "ctrl+r":
		m.req.Filter = search.NextFilter(m.req.Filter)
		m.searching = true
		return m, m.runSearch()
	case "ctrl+s":
		m.req.Mode = search.NextMode(m.req.Mode)
		m.searching = true
		return m, m.runSearch()
	case "ctrl+o":
		if len(m.rows) > 0 {
			m.inspector = true
		}
		return m, nil
	case "alt+1", "alt+2", "alt+3", "alt+4", "alt+5", "alt+6", "alt+7", "alt+8", "alt+9":
		idx := int(msg.String()[len("alt+")] - '1')
		if idx >= 0 && idx < len(m.rows) {
			m.cursor = idx
			return m, m.loadPreview(m.rows[idx].EventID)
		}
		return m, nil
	default:
		prev := m.input.Value()
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		if m.input.Value() != prev {
			m.searching = true
			return m, tea.Batch(cmd, m.runSearch())
		}
		return m, cmd
	}
}

func (m model) currentEventID() int64 {
	if m.cursor < 0 || m.cursor >= len(m.rows) {
		return 0
	}
	return m.rows[m.cursor].EventID
}

func (m model) inspectorText() string {
	if m.preview != "" {
		return m.preview
	}
	if len(m.rows) == 0 {
		return "(no selection)"
	}
	return fmt.Sprintf("event_id: %d\ncommand:  %s", m.rows[m.cursor].EventID, m.rows[m.cursor].Cmd)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
