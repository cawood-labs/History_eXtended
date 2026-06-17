package tui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mrcawood/History_eXtended/internal/search"
)

// Package-level styles are rebound in initStyles before each Run. Lipgloss
// detects color from its renderer's writer; zsh widgets pipe stdout so the
// default renderer (stdout) would disable ANSI color.
var (
	styleTitle   lipgloss.Style
	styleMuted   lipgloss.Style
	styleSel     lipgloss.Style
	styleExitOK  lipgloss.Style
	styleExitBad lipgloss.Style
	styleSync    lipgloss.Style
	styleFooter  lipgloss.Style
	stylePreview lipgloss.Style
)

func initStyles(w io.Writer) {
	r := lipgloss.NewRenderer(w)
	lipgloss.SetDefaultRenderer(r)
	styleTitle = r.NewStyle().Bold(true).Foreground(lipgloss.Color("69"))
	styleMuted = r.NewStyle().Foreground(lipgloss.Color("241"))
	styleSel = r.NewStyle().Bold(true).Foreground(lipgloss.Color("136")) // dark yellow
	styleExitOK = r.NewStyle().Foreground(lipgloss.Color("42"))
	styleExitBad = r.NewStyle().Foreground(lipgloss.Color("196"))
	styleSync = r.NewStyle().Foreground(lipgloss.Color("214"))
	styleFooter = r.NewStyle().Foreground(lipgloss.Color("241"))
	stylePreview = r.NewStyle().Padding(0, 1)
}

func (m model) View() string {
	if m.width == 0 {
		m.width = 80
	}
	if m.inspector {
		return m.viewInspector()
	}

	header := styleTitle.Render(fmt.Sprintf("filter: %s  mode: %s", search.FilterName(m.req.Filter), search.ModeName(m.req.Mode)))
	if m.searching {
		header += styleMuted.Render("  searching…")
	}

	listW := m.width*3/5 - 2
	if listW < 20 {
		listW = m.width - 4
	}
	prevW := m.width - listW - 4
	if prevW < 10 {
		prevW = 0
	}

	listPane := m.renderList(listW)
	var body string
	if prevW > 0 {
		prevPane := stylePreview.Width(prevW).Render(m.renderPreview(prevW))
		body = lipgloss.JoinHorizontal(lipgloss.Top, listPane, prevPane)
	} else {
		body = listPane
	}

	enterAction, tabAction := "edit", "run"
	if m.enterAccept {
		enterAction, tabAction = "run", "edit"
	}
	footer := styleFooter.Render(fmt.Sprintf("Enter %s · Tab %s · Ctrl-R filter · Ctrl-S mode · Ctrl-O inspector · Esc cancel", enterAction, tabAction))
	if m.inline {
		h := m.inlineHeight
		if h <= 0 {
			h = 15
		}
		content := lipgloss.JoinVertical(lipgloss.Left, header, body, m.input.View(), footer)
		return lipgloss.NewStyle().MaxHeight(h).Render(content)
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		body,
		m.input.View(),
		footer,
	)
}

func (m model) viewInspector() string {
	title := styleTitle.Render("Inspector — Esc/Ctrl-O close")
	body := stylePreview.Render(m.inspectorText())
	return lipgloss.JoinVertical(lipgloss.Left, title, body)
}

func (m model) renderList(width int) string {
	if len(m.rows) == 0 {
		return styleMuted.Width(width).Render("(no matches)")
	}
	var b strings.Builder
	visible := m.visibleRows()
	for i, row := range visible.rows {
		selected := i == m.cursor-visible.offset
		line := m.formatRow(row, width-2, selected)
		if selected {
			line = styleSel.Width(width).Render(line)
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}

type visibleWindow struct {
	rows   []search.Row
	offset int
}

func (m model) visibleRows() visibleWindow {
	maxRows := m.listHeight()
	if maxRows < 1 {
		maxRows = 10
	}
	if len(m.rows) <= maxRows {
		return visibleWindow{rows: m.rows, offset: 0}
	}
	offset := m.cursor - maxRows/2
	if offset < 0 {
		offset = 0
	}
	if offset+maxRows > len(m.rows) {
		offset = len(m.rows) - maxRows
	}
	return visibleWindow{rows: m.rows[offset : offset+maxRows], offset: offset}
}

func (m model) listHeight() int {
	if m.inline {
		h := m.inlineHeight - 6
		if h < 3 {
			return 3
		}
		return h
	}
	h := m.height - 8
	if h < 5 {
		return 5
	}
	return h
}

func (m model) formatRow(r search.Row, width int, selected bool) string {
	exit := "-"
	if r.ExitCode != nil {
		exit = fmt.Sprintf("%d", *r.ExitCode)
	}
	when := search.RelTime(r.StartedAt)
	host := r.Host
	dup := ""
	if r.DupCount > 1 {
		dup = fmt.Sprintf(" ×%d", r.DupCount)
	}

	// Nested lipgloss styles emit reset codes that break selection backgrounds;
	// use plain text on the highlighted row.
	if selected {
		cmd := r.Cmd
		maxCmd := width - len(exit) - len(when) - len(host) - len(dup) - 6
		if maxCmd < 10 {
			maxCmd = 10
		}
		if len(cmd) > maxCmd {
			cmd = cmd[:maxCmd-3] + "..."
		}
		return fmt.Sprintf("%s %s %s  %s%s", exit, when, host, cmd, dup)
	}

	exitStyle := styleMuted
	if r.ExitCode != nil {
		if *r.ExitCode == 0 {
			exitStyle = styleExitOK
		} else {
			exitStyle = styleExitBad
		}
	}
	if r.Origin == "sync" && host != "" {
		host = styleSync.Render(host)
	}
	if r.DupCount > 1 {
		dup = styleMuted.Render(fmt.Sprintf(" ×%d", r.DupCount))
	}
	meta := fmt.Sprintf("%s %s %s", exitStyle.Render(exit), when, host)
	cmd := r.Cmd
	maxCmd := width - lipgloss.Width(meta) - 2
	if maxCmd < 10 {
		maxCmd = 10
	}
	if len(cmd) > maxCmd {
		cmd = cmd[:maxCmd-3] + "..."
	}
	return meta + "  " + cmd + dup
}

func (m model) renderPreview(width int) string {
	if m.preview == "" {
		return styleMuted.Render("…")
	}
	lines := strings.Split(m.preview, "\n")
	var out []string
	for _, ln := range lines {
		if len(ln) > width {
			ln = ln[:width-3] + "..."
		}
		out = append(out, ln)
	}
	return strings.Join(out, "\n")
}
