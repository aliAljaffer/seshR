package main

import (
	"bufio"
	"database/sql"
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

	_ "modernc.org/sqlite"
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
		if alive(r.PID, time.UnixMilli(r.StartedAt)) {
			st = "open"
		}
		t := r.UpdatedAt
		if t == 0 {
			t = r.StartedAt
		}
		seen[r.SessionID] = true
		out = append(out, sess{"claude", r.SessionID, r.Cwd, titleOf(home, r.SessionID, r.Name), st, time.UnixMilli(t)})
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
			out = append(out, sess{"claude", id, cwd, titleOf(home, id, ""), "recent", fi.ModTime()})
		}
	}
	return out
}

// titleOf returns the AI title, else the last prompt, else fallback. Both are
// rewritten as the session grows, so the file tail is enough.
func titleOf(home, id, fallback string) string {
	files, _ := filepath.Glob(filepath.Join(home, ".claude/projects/*", id+".jsonl"))
	if len(files) == 0 {
		return fallback
	}
	fh, err := os.Open(files[0])
	if err != nil {
		return fallback
	}
	defer fh.Close()
	const tail = 256 << 10
	buf := make([]byte, tail)
	n, _ := fh.ReadAt(buf, max(0, fileSize(fh)-tail))
	var title, prompt string
	for _, l := range strings.Split(string(buf[:n]), "\n") {
		var e struct{ AiTitle, LastPrompt string }
		if strings.HasPrefix(l, `{"type":"ai-title"`) || strings.HasPrefix(l, `{"type":"last-prompt"`) {
			if json.Unmarshal([]byte(l), &e) == nil {
				if e.AiTitle != "" {
					title = e.AiTitle
				}
				if e.LastPrompt != "" {
					prompt = e.LastPrompt
				}
			}
		}
	}
	switch {
	case title != "":
		return title
	case prompt != "":
		return prompt
	}
	return fallback
}

func fileSize(f *os.File) int64 {
	fi, err := f.Stat()
	if err != nil {
		return 0
	}
	return fi.Size()
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
	p := filepath.Join(home, ".local/share/opencode/opencode.db")
	if _, err := os.Stat(p); err != nil {
		return nil
	}
	db, err := sql.Open("sqlite", "file:"+p+"?mode=ro&_pragma=busy_timeout(2000)")
	if err != nil {
		return nil
	}
	defer db.Close()
	// OpenCode 2.x renamed the table; older installs still use "session".
	tbl := "session"
	var n int
	if db.QueryRow("select count(*) from sqlite_master where type='table' and name='session_v2'").Scan(&n) == nil && n > 0 {
		tbl = "session_v2"
	}
	rows, err := db.Query("select id, directory, coalesce(title, ''), time_updated from "+tbl+" where parent_id is null and time_archived is null and time_updated > ?", since.UnixMilli())
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []sess
	for rows.Next() {
		var id, dir, title string
		var t int64
		if rows.Scan(&id, &dir, &title, &t) == nil {
			out = append(out, sess{"opencode", id, dir, title, "recent", time.UnixMilli(t)})
		}
	}
	return out
}

func main() {
	win := flag.Duration("since", 24*time.Hour, "include recent (non-registry) sessions newer than this")
	here := flag.Bool("here", false, "only sessions in the current directory")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: seshr [-since 24h] [--here]   list sessions\n       seshr N              resume session N in its directory")
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

	cwd, _ := os.Getwd()
	for i, s := range all {
		if *here && filepath.Clean(s.Dir) != cwd {
			continue
		}
		fmt.Printf("%2d  %-8s %-7s %-5s %s  %s\n", i+1, s.Tool, s.State, ago(s.When), strings.Replace(s.Dir, home, "~", 1), label(s))
	}
}

func label(s sess) string {
	if s.Name == "" {
		return s.ID[:8]
	}
	n := strings.Join(strings.Fields(s.Name), " ")
	if r := []rune(n); len(r) > 70 {
		return string(r[:69]) + "…"
	}
	return n
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
