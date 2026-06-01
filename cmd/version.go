package cmd

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

// Version is set at build time via -ldflags.
var Version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print op-cli version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("op-cli %s (%s/%s)\n", Version, runtime.GOOS, runtime.GOARCH)
	},
}

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade op-cli to the latest version",
	Long: `Download and install the latest op-cli binary.

Uses glab CLI (preferred) or GITLAB_TOKEN to download from the package registry.`,
	RunE: runUpgrade,
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(upgradeCmd)
}

const pkgBaseURL = "https://gitlab-tw.ddns.net/api/v4/projects/gmedtn%2Fop-cli/packages/generic/op-cli/latest"

func runUpgrade(cmd *cobra.Command, args []string) error {
	binary := fmt.Sprintf("op-%s-%s", runtime.GOOS, runtime.GOARCH)

	fmt.Printf("Current: op-cli %s\n", Version)
	fmt.Printf("Downloading latest %s...\n", binary)

	// Find where the current binary lives so we can replace it.
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding current binary: %w", err)
	}
	execPath, _ = filepath.EvalSymlinks(execPath)

	// Try glab first, then curl with token.
	tmpPath, err := downloadViaGlab(binary)
	if err != nil {
		tmpPath, err = downloadViaCurl(binary)
	}
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

// downloadViaGlab uses the glab CLI to download the release asset.
func downloadViaGlab(assetName string) (string, error) {
	if _, err := exec.LookPath("glab"); err != nil {
		return "", fmt.Errorf("glab not found")
	}

	dir, err := os.MkdirTemp("", "op-upgrade-*")
	if err != nil {
		return "", err
	}

	cmd := exec.Command("glab", "release", "download",
		"--repo", "gmedtn/op-cli",
		"--include-external",
		"--asset-name="+assetName,
		"-D", dir,
	)
	cmd.Env = append(os.Environ(), "GITLAB_HOST=gitlab-tw.ddns.net")

	out, err := cmd.CombinedOutput()
	if err != nil {
		os.RemoveAll(dir)
		return "", fmt.Errorf("glab download failed: %s", strings.TrimSpace(string(out)))
	}

	path := filepath.Join(dir, assetName)
	if _, err := os.Stat(path); err != nil {
		os.RemoveAll(dir)
		return "", fmt.Errorf("asset not found after glab download")
	}

	// Move out of temp dir so the dir can be cleaned up independently.
	tmp, err := os.CreateTemp("", "op-bin-*")
	if err != nil {
		os.RemoveAll(dir)
		return "", err
	}
	tmpPath := tmp.Name()
	tmp.Close()

	if err := copyFile(path, tmpPath); err != nil {
		os.RemoveAll(dir)
		os.Remove(tmpPath)
		return "", err
	}
	os.RemoveAll(dir)

	fmt.Println("Downloaded via glab.")
	return tmpPath, nil
}

// downloadViaCurl downloads the binary using a GitLab token.
func downloadViaCurl(assetName string) (string, error) {
	token := os.Getenv("GITLAB_TOKEN")
	if token == "" {
		return "", fmt.Errorf(
			"could not download. Try one of:\n" +
				"  1. Authenticate glab:  GITLAB_HOST=gitlab-tw.ddns.net glab auth login\n" +
				"  2. Set token:          export GITLAB_TOKEN=your-token")
	}

	url := pkgBaseURL + "/" + assetName

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("PRIVATE-TOKEN", token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("downloading: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("download failed (HTTP %d). Check your GITLAB_TOKEN", resp.StatusCode)
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

	fmt.Println("Downloaded via token.")
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
