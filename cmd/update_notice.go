package cmd

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// latestReleaseURL redirects to the newest release's tag page. A var so tests
// can point it at a local server.
var latestReleaseURL = "https://github.com/VacronBruce/op-cli/releases/latest"

// updateCacheFile is where the newest release tag is remembered for a day.
var updateCacheFile = func() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "op-cli", "latest-release")
}

// printUpdateNotice tells the user, on stderr, that a newer op-cli exists.
func printUpdateNotice() {
	if tag := newerRelease(Version, updateCacheFile(), fetchLatestTag); tag != "" {
		fmt.Fprintf(os.Stderr, "op-cli %s is out (this is %s); update with: op upgrade\n", tag, Version)
	}
}

// newerRelease returns the newest release tag when it is newer than current.
// It asks GitHub at most once a day, failed attempts included, and stays
// quiet for dev builds and when OP_NO_UPDATE_CHECK is set.
func newerRelease(current, cacheFile string, fetch func() (string, error)) string {
	have, ok := parseVersion(current)
	if !ok || cacheFile == "" || os.Getenv("OP_NO_UPDATE_CHECK") != "" {
		return ""
	}
	data, readErr := os.ReadFile(cacheFile)
	tag := strings.TrimSpace(string(data))
	if info, err := os.Stat(cacheFile); readErr != nil || err != nil || time.Since(info.ModTime()) >= 24*time.Hour {
		if fetched, err := fetch(); err == nil {
			tag = fetched
		}
		_ = os.MkdirAll(filepath.Dir(cacheFile), 0o755)
		_ = os.WriteFile(cacheFile, []byte(tag+"\n"), 0o644)
	}
	latest, ok := parseVersion(tag)
	if !ok {
		return ""
	}
	for i := range latest {
		if latest[i] != have[i] {
			if latest[i] > have[i] {
				return tag
			}
			return ""
		}
	}
	return ""
}

// parseVersion reads major.minor.patch from "v0.27.0", "0.27.0" or a build
// like "0.25.0-13-gd5e1601", ignoring what follows the dash.
func parseVersion(v string) ([3]int, bool) {
	var parts [3]int
	core, _, _ := strings.Cut(strings.TrimPrefix(v, "v"), "-")
	fields := strings.Split(core, ".")
	if len(fields) != 3 {
		return parts, false
	}
	for i, f := range fields {
		n, err := strconv.Atoi(f)
		if err != nil {
			return parts, false
		}
		parts[i] = n
	}
	return parts, true
}

// fetchLatestTag reads the newest release tag from the redirect GitHub sends
// for /releases/latest, which needs no API token and no rate limit.
func fetchLatestTag() (string, error) {
	c := &http.Client{
		Timeout: 2 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := c.Head(latestReleaseURL)
	if err != nil {
		return "", err
	}
	resp.Body.Close()
	location := resp.Header.Get("Location")
	if !strings.Contains(location, "/releases/tag/") {
		return "", fmt.Errorf("no release tag in the redirect to %q", location)
	}
	return path.Base(location), nil
}
