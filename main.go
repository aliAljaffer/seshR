package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type sess struct {
	Tool, ID, Dir, Name, State string
	When                       time.Time
}

func (s sess) cmd() []string {
	if s.Tool == "claude" {
		return []string{"claude", "--resume", s.ID}
	}
	return []string{"opencode", "-s", s.ID}
}

func alive(pid int) bool {
	out, err := exec.Command("ps", "-o", "command=", "-p", strconv.Itoa(pid)).Output()
	// A reused PID runs some other command.
	return err == nil && strings.Contains(string(out), "claude")
}

func claude(home string, since time.Time) []sess {
	var out []sess
	seen := map[string]bool{}
	files, _ := filepath.Glob(filepath.Join(home, ".claude/sessions/*.json"))
	for _, f := range files {
		b, _ := os.ReadFile(f)
		var r struct {
			PID       int
			SessionID string `json:"sessionId"`
			Cwd, Name string
			StartedAt int64 `json:"startedAt"`
			UpdatedAt int64 `json:"updatedAt"`
		}
		if json.Unmarshal(b, &r) != nil || r.SessionID == "" {
			continue
		}
		st := "crashed"
		if alive(r.PID) {
			st = "open"
		}
		t := r.UpdatedAt
		if t == 0 {
			t = r.StartedAt
		}
		seen[r.SessionID] = true
		out = append(out, sess{"claude", r.SessionID, r.Cwd, r.Name, st, time.UnixMilli(t)})
	}
	// Fallback in case Claude Code prunes stale registry files: recent transcripts.
	jl, _ := filepath.Glob(filepath.Join(home, ".claude/projects/*/*.jsonl"))
	for _, f := range jl {
		id := strings.TrimSuffix(filepath.Base(f), ".jsonl")
		fi, err := os.Stat(f)
		if seen[id] || err != nil || fi.ModTime().Before(since) {
			continue
		}
		if cwd := transcriptCwd(f); cwd != "" {
			out = append(out, sess{"claude", id, cwd, "", "recent", fi.ModTime()})
		}
	}
	return out
}

func transcriptCwd(f string) string {
	fh, err := os.Open(f)
	if err != nil {
		return ""
	}
	defer fh.Close()
	sc := bufio.NewScanner(fh)
	sc.Buffer(make([]byte, 1<<20), 16<<20)
	for i := 0; i < 50 && sc.Scan(); i++ {
		var l struct{ Cwd string }
		if json.Unmarshal(sc.Bytes(), &l) == nil && l.Cwd != "" {
			return l.Cwd
		}
	}
	return ""
}

func opencode(home string, since time.Time) []sess {
	db := filepath.Join(home, ".local/share/opencode/opencode.db")
	if _, err := os.Stat(db); err != nil {
		return nil
	}
	q := fmt.Sprintf("select id, directory, title, time_updated as t from session where parent_id is null and time_archived is null and time_updated > %d", since.UnixMilli())
	b, err := exec.Command("sqlite3", "-readonly", "-json", db, q).Output()
	if err != nil || len(b) == 0 {
		return nil
	}
	var rows []struct {
		ID, Directory, Title string
		T                    int64 `json:"t"`
	}
	if json.Unmarshal(b, &rows) != nil {
		return nil
	}
	var out []sess
	for _, r := range rows {
		out = append(out, sess{"opencode", r.ID, r.Directory, r.Title, "recent", time.UnixMilli(r.T)})
	}
	return out
}

func main() {
	win := flag.Duration("since", 24*time.Hour, "include recent (non-registry) sessions newer than this")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: seshr [-since 24h]   list sessions\n       seshr N              resume session N in its directory")
	}
	flag.Parse()
	home, _ := os.UserHomeDir()
	since := time.Now().Add(-*win)
	all := append(claude(home, since), opencode(home, since)...)
	sort.Slice(all, func(i, j int) bool { return all[i].When.After(all[j].When) })

	if flag.NArg() == 1 {
		n, err := strconv.Atoi(flag.Arg(0))
		if err != nil || n < 1 || n > len(all) {
			fmt.Fprintln(os.Stderr, "no such session number")
			os.Exit(1)
		}
		s := all[n-1]
		if err := os.Chdir(s.Dir); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		c := s.cmd()
		p, err := exec.LookPath(c[0])
		if err == nil {
			err = syscall.Exec(p, c, os.Environ())
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	for i, s := range all {
		fmt.Printf("%2d  %-8s %-7s %-5s %s  %s\n", i+1, s.Tool, s.State, ago(s.When), strings.Replace(s.Dir, home, "~", 1), label(s))
	}
}

func label(s sess) string {
	if s.Name != "" {
		return s.Name
	}
	return s.ID[:8]
}

func ago(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 48*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}
