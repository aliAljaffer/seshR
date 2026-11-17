package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	repo        = "alialjaffer/seshr"
	brewFormula = "alialjaffer/tap/seshr"
)

var (
	version     = "dev"
	releasesAPI = "https://api.github.com/repos/" + repo + "/releases/latest"
	downloadURL = "https://github.com/" + repo + "/releases/download"
	httpClient  = &http.Client{Timeout: 2 * time.Minute}
)

func upgrade(args []string) error {
	fs := flag.NewFlagSet("upgrade", flag.ExitOnError)
	yes := fs.Bool("yes", false, "upgrade without asking for confirmation")
	fs.BoolVar(yes, "y", false, "same as --yes")
	fs.Parse(args)

	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}

	tag, err := latestTag()
	if err != nil {
		return fmt.Errorf("checking latest release: %w", err)
	}
	latest := strings.TrimPrefix(tag, "v")
	if version != "dev" && !isNewer(latest, version) {
		fmt.Printf("seshr %s is up to date.\n", version)
		return nil
	}

	fmt.Printf("New version found! %s -> %s\n", version, latest)
	if !*yes {
		if fi, err := os.Stdin.Stat(); err != nil || fi.Mode()&os.ModeCharDevice == 0 {
			return fmt.Errorf("not a terminal; pass --yes to upgrade without a prompt")
		}
		fmt.Print("Upgrade? [y/N] ")
		answer, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		if a := strings.ToLower(strings.TrimSpace(answer)); a != "y" && a != "yes" {
			return nil
		}
	}

	if strings.Contains(filepath.ToSlash(exe), "/Cellar/") {
		c := exec.Command("brew", "upgrade", brewFormula)
		c.Stdout, c.Stderr = os.Stdout, os.Stderr
		return c.Run()
	}
	if err := downloadAndReplace(tag, exe); err != nil {
		return err
	}
	fmt.Printf("Upgraded to %s.\n", latest)
	return nil
}

func latestTag() (string, error) {
	resp, err := httpClient.Get(releasesAPI)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GET %s: %s", releasesAPI, resp.Status)
	}
	var r struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil || r.TagName == "" {
		return "", fmt.Errorf("unexpected response from %s", releasesAPI)
	}
	return r.TagName, nil
}

func isNewer(a, b string) bool {
	pa, pb := versionParts(a), versionParts(b)
	if pa == nil || pb == nil {
		return false
	}
	for i := 0; i < len(pa) || i < len(pb); i++ {
		var x, y int
		if i < len(pa) {
			x = pa[i]
		}
		if i < len(pb) {
			y = pb[i]
		}
		if x != y {
			return x > y
		}
	}
	return false
}

func versionParts(v string) []int {
	core, _, _ := strings.Cut(strings.TrimPrefix(v, "v"), "-")
	var parts []int
	for _, s := range strings.Split(core, ".") {
		n, err := strconv.Atoi(s)
		if err != nil {
			return nil
		}
		parts = append(parts, n)
	}
	return parts
}

func downloadAndReplace(tag, exe string) error {
	asset := "seshr-" + runtime.GOOS + "-" + runtime.GOARCH
	body, err := httpGet(downloadURL + "/" + tag + "/" + asset)
	if err != nil {
		return fmt.Errorf("no release binary for %s/%s: %w", runtime.GOOS, runtime.GOARCH, err)
	}
	sums, err := httpGet(downloadURL + "/" + tag + "/checksums.txt")
	if err != nil {
		return err
	}
	want := ""
	for _, line := range strings.Split(string(sums), "\n") {
		if f := strings.Fields(line); len(f) == 2 && f[1] == asset {
			want = f[0]
		}
	}
	got := sha256.Sum256(body)
	if want == "" || hex.EncodeToString(got[:]) != want {
		return fmt.Errorf("checksum mismatch for %s; refusing to install", asset)
	}

	tmp, err := os.CreateTemp(filepath.Dir(exe), ".seshr-upgrade-*")
	if err != nil {
		return fmt.Errorf("cannot write next to %s (try sudo): %w", exe, err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(body); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), 0o755); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), exe)
}

func httpGet(url string) ([]byte, error) {
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	return io.ReadAll(resp.Body)
}
