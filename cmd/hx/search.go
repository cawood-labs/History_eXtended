package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/mrcawood/History_eXtended/internal/db"
	"github.com/mrcawood/History_eXtended/internal/search"
	"github.com/mrcawood/History_eXtended/internal/tui"
	"golang.org/x/term"
)

// searchExitRun is the exit code from `hx search -i` meaning "run the selected
// command immediately" (enter_accept). The shell widget maps it to accept-line.
const searchExitRun = 10

type searchOpts struct {
	filter      string
	mode        string
	format      string
	limit       int
	dedup       bool
	noDedup     bool
	noImport    bool
	interactive bool
	query       string
}

func cmdSearch(args []string) {
	opts, err := parseSearchArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "hx search: %v\n", err)
		os.Exit(1)
	}

	conn, err := db.Open(dbPath())
	if err != nil {
		fmt.Fprintf(os.Stderr, "hx search: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = conn.Close() }()

	cfg := getConfig()
	filter, err := search.ParseFilter(opts.filter)
	if err != nil {
		fmt.Fprintf(os.Stderr, "hx search: %v\n", err)
		os.Exit(1)
	}
	mode, err := search.ParseMode(opts.mode)
	if err != nil {
		fmt.Fprintf(os.Stderr, "hx search: %v\n", err)
		os.Exit(1)
	}

	req := search.Request{
		Query:     opts.query,
		Filter:    filter,
		Mode:      mode,
		Host:      searchEnvHost(),
		Cwd:       searchEnvCwd(),
		SessionID: os.Getenv("HX_SESSION_ID"),
		Dedup:     opts.dedup,
		Limit:     opts.limit,
		NoImport:  opts.noImport,
	}

	if opts.interactive {
		if !term.IsTerminal(int(os.Stdin.Fd())) {
			fmt.Fprintf(os.Stderr, "hx search: %v\n", tui.ErrNotTTY)
			os.Exit(1)
		}
		res, err := tui.Run(tui.Options{Conn: conn, Cfg: cfg, Req: req})
		if err != nil {
			fmt.Fprintf(os.Stderr, "hx search: %v\n", err)
			os.Exit(1)
		}
		if res.Cancelled || res.Cmd == "" {
			return
		}
		fmt.Println(res.Cmd)
		if res.RunRequested {
			// Exit 10 signals the shell widget to run (accept-line) rather than
			// just insert the command for editing. os.Exit skips defers, so close.
			_ = conn.Close()
			os.Exit(searchExitRun)
		}
		return
	}

	format, err := search.ParseFormat(opts.format)
	if err != nil {
		fmt.Fprintf(os.Stderr, "hx search: %v\n", err)
		os.Exit(1)
	}

	rows, err := search.Search(context.Background(), conn, cfg, req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "hx search: %v\n", err)
		os.Exit(1)
	}
	if err := search.WriteRows(os.Stdout, format, rows); err != nil {
		fmt.Fprintf(os.Stderr, "hx search: %v\n", err)
		os.Exit(1)
	}
}

func cmdShow(args []string) {
	var eventID int64
	raw := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--raw":
			raw = true
		case "-h", "--help":
			printSubcommandHelp(os.Stdout, "show")
			return
		default:
			if strings.HasPrefix(args[i], "-") {
				fmt.Fprintf(os.Stderr, "hx show: unknown flag %s\n", args[i])
				os.Exit(1)
			}
			var err error
			eventID, err = parseEventID(args[i])
			if err != nil {
				fmt.Fprintf(os.Stderr, "hx show: %v\n", err)
				os.Exit(1)
			}
		}
	}
	if eventID == 0 {
		fmt.Fprintf(os.Stderr, "hx show: usage: hx show [--raw] <event_id>\n")
		os.Exit(1)
	}

	conn, err := db.Open(dbPath())
	if err != nil {
		fmt.Fprintf(os.Stderr, "hx show: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = conn.Close() }()

	d, err := search.GetEvent(conn, eventID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "hx show: %v\n", err)
		os.Exit(1)
	}
	if raw {
		fmt.Println(d.Cmd)
		return
	}
	fmt.Println(search.FormatDetail(d))
}

func parseEventID(s string) (int64, error) {
	var id int64
	_, err := fmt.Sscanf(s, "%d", &id)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid event_id %q", s)
	}
	return id, nil
}

func parseSearchArgs(args []string) (searchOpts, error) {
	opts := searchOpts{
		limit: 50,
		dedup: true,
	}
	cfg := getConfig()
	if cfg != nil {
		if cfg.Search.DefaultFilter != "" {
			opts.filter = cfg.Search.DefaultFilter
		}
		if cfg.Search.DefaultMode != "" {
			opts.mode = cfg.Search.DefaultMode
		}
	}
	var queryParts []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--filter":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("--filter requires value")
			}
			opts.filter = args[i+1]
			i++
		case "--mode":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("--mode requires value")
			}
			opts.mode = args[i+1]
			i++
		case "--format":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("--format requires value")
			}
			opts.format = args[i+1]
			i++
		case "--limit":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("--limit requires value")
			}
			if _, err := fmt.Sscanf(args[i+1], "%d", &opts.limit); err != nil {
				return opts, fmt.Errorf("invalid --limit")
			}
			i++
		case "--no-dedup":
			opts.noDedup = true
		case "--no-import":
			opts.noImport = true
		case "-i", "--interactive":
			opts.interactive = true
		case "-h", "--help":
			printSubcommandHelp(os.Stdout, "search")
			os.Exit(0)
		default:
			if strings.HasPrefix(args[i], "-") {
				return opts, fmt.Errorf("unknown flag %s", args[i])
			}
			queryParts = append(queryParts, args[i])
		}
	}
	if opts.noDedup {
		opts.dedup = false
	}
	if opts.filter == "" {
		opts.filter = "global"
	}
	if opts.mode == "" {
		opts.mode = "fuzzy"
	}
	if opts.format == "" {
		opts.format = "table"
	}
	opts.query = strings.TrimSpace(strings.Join(queryParts, " "))
	return opts, nil
}

func searchEnvHost() string {
	if v := os.Getenv("HX_SEARCH_HOST"); v != "" {
		return v
	}
	out, err := exec.Command("hostname").Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

func searchEnvCwd() string {
	if v := os.Getenv("HX_SEARCH_CWD"); v != "" {
		return v
	}
	if v := os.Getenv("PWD"); v != "" {
		return v
	}
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return wd
}
