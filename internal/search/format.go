package search

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
)

const (
	FieldSep  = "\x1f"
	RecordSep = "\x00"
)

// FormatKind is machine/human output format for search results.
type FormatKind int

const (
	FormatTable FormatKind = iota
	FormatTSV
	FormatNull
	FormatJSON
)

// ParseFormat parses --format value.
func ParseFormat(s string) (FormatKind, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "table":
		return FormatTable, nil
	case "tsv":
		return FormatTSV, nil
	case "null", "nul":
		return FormatNull, nil
	case "json":
		return FormatJSON, nil
	default:
		return FormatTable, fmt.Errorf("unknown format %q", s)
	}
}

// WriteRows emits search results in the requested format.
func WriteRows(w io.Writer, format FormatKind, rows []Row) error {
	switch format {
	case FormatNull:
		return writeNull(w, rows)
	case FormatTSV:
		return writeTSV(w, rows)
	case FormatJSON:
		return writeJSON(w, rows)
	default:
		return writeTable(w, rows)
	}
}

func exitStr(exit *int) string {
	if exit == nil {
		return "-"
	}
	return strconv.Itoa(*exit)
}

func writeNull(w io.Writer, rows []Row) error {
	for _, r := range rows {
		line := strings.Join([]string{
			strconv.FormatInt(r.EventID, 10),
			r.Cmd,
			exitStr(r.ExitCode),
			RelTime(r.StartedAt),
			r.Cwd,
			strconv.Itoa(r.DupCount),
		}, FieldSep)
		if _, err := io.WriteString(w, line+RecordSep); err != nil {
			return err
		}
	}
	return nil
}

func writeTSV(w io.Writer, rows []Row) error {
	for _, r := range rows {
		_, err := fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%d\n",
			r.EventID, r.Cmd, exitStr(r.ExitCode), RelTime(r.StartedAt), r.Cwd, r.DupCount)
		if err != nil {
			return err
		}
	}
	return nil
}

func writeJSON(w io.Writer, rows []Row) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(rows)
}

func writeTable(w io.Writer, rows []Row) error {
	if len(rows) == 0 {
		_, err := fmt.Fprintln(w, "(no matches)")
		return err
	}
	_, _ = fmt.Fprintf(w, "%-8s %-5s %-8s %-18s %s\n", "id", "exit", "when", "cwd", "cmd")
	for _, r := range rows {
		cwd := r.Cwd
		if len(cwd) > 18 {
			cwd = cwd[:15] + "..."
		}
		_, err := fmt.Fprintf(w, "%-8d %-5s %-8s %-18s %s\n",
			r.EventID, exitStr(r.ExitCode), RelTime(r.StartedAt), cwd, r.Cmd)
		if err != nil {
			return err
		}
	}
	return nil
}
