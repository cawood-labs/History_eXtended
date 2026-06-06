package tui

import "github.com/mrcawood/History_eXtended/internal/search"

type searchDoneMsg struct {
	rows []search.Row
	err  error
}

type detailDoneMsg struct {
	text string
	err  error
}
