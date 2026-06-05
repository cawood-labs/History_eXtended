package search

import "strings"

// IsSelfCmd reports whether cmd is hx querying itself (exclude from interactive search).
func IsSelfCmd(cmd string) bool {
	cmd = strings.TrimSpace(cmd)
	if cmd == "hx" || strings.HasPrefix(cmd, "hx ") {
		return true
	}
	if cmd == "./bin/hx" || strings.HasPrefix(cmd, "./bin/hx ") {
		return true
	}
	if strings.Contains(cmd, "| hx ") || strings.Contains(cmd, "| hx") {
		return true
	}
	if strings.Contains(cmd, " hx ") || strings.HasSuffix(cmd, " hx") {
		return true
	}
	return false
}
