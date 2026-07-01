package cmd

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"
)

// Version is set at build time via -ldflags.
var Version = "dev"

var versionCmd = &cobra.Command{
	Use:         "version",
	Short:       "Print op-cli version",
	Annotations: skipClientInit(),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("op-cli %s (%s/%s)\n", Version, runtime.GOOS, runtime.GOARCH)
	},
}

var upgradeCmd = &cobra.Command{
	Use:         "upgrade",
	Short:       "Upgrade op-cli to the latest version",
	Annotations: skipClientInit(),
	Long: `Download and install the latest op-cli binary.

Downloads the newest release asset from the public GitHub repo — no auth needed.`,
	RunE: runUpgrade,
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(upgradeCmd)
}

// releaseBaseURL is a var (not const) so tests can point it at a local server.
// GitHub serves the newest release at this stable path; the repo is public so
// the asset is a plain download with no token.
var releaseBaseURL = "https://github.com/VacronBruce/op-cli/releases/latest/download"

// Seams for tests. Production wiring locates the running binary and downloads
// the release asset — both do network/file I/O that tests stub out to cover
// runUpgrade's orchestration.
var (
	upgradeExecPath = func() (string, error) {
		p, err := os.Executable()
		if err != nil {
			return "", err
		}
		p, _ = filepath.EvalSymlinks(p)
		return p, nil
	}
	upgradeDownload = downloadRelease
)

func runUpgrade(cmd *cobra.Command, args []string) error {
	binary := fmt.Sprintf("op-%s-%s", runtime.GOOS, runtime.GOARCH)

	fmt.Printf("Current: op-cli %s\n", Version)
	fmt.Printf("Downloading latest %s...\n", binary)

	// Find where the current binary lives so we can replace it.
	execPath, err := upgradeExecPath()
	if err != nil {
		return fmt.Errorf("finding current binary: %w", err)
	}

	tmpPath, err := upgradeDownload(binary)
	if err != nil {
		return err
	}
	defer os.Remove(tmpPath) // clean up on any exit path

	if err := os.Chmod(tmpPath, 0755); err != nil {
		return fmt.Errorf("setting permissions: %w", err)
	}

	// Replace current binary.
	if err := os.Rename(tmpPath, execPath); err != nil {
		// Rename fails across filesystems; fall back to copy.
		if copyErr := copyFile(tmpPath, execPath); copyErr != nil {
			return fmt.Errorf("replacing binary: %w (rename: %w)", copyErr, err)
		}
	}

	fmt.Printf("Upgraded: %s\n", execPath)
	fmt.Println("Run 'op version' to verify.")
	return nil
}

// downloadRelease fetches the release asset from the public GitHub repo.
// No auth: the repo is public, so /releases/latest/download/<asset> is a plain GET.
func downloadRelease(assetName string) (string, error) {
	url := releaseBaseURL + "/" + assetName

	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("downloading: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("download failed (HTTP %d) for %s", resp.StatusCode, url)
	}

	tmp, err := os.CreateTemp("", "op-bin-*")
	if err != nil {
		return "", fmt.Errorf("creating temp file: %w", err)
	}

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return "", fmt.Errorf("writing binary: %w", err)
	}
	tmp.Close()

	fmt.Println("Downloaded.")
	return tmp.Name(), nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
