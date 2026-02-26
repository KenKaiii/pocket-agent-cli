package video

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/unstablemind/pocket/pkg/output"
)

// DownloadResult is the LLM-friendly download result
type DownloadResult struct {
	Title    string  `json:"title"`
	Filename string  `json:"filename"`
	URL      string  `json:"url"`
	Duration float64 `json:"duration,omitempty"`
	Filesize int64   `json:"filesize,omitempty"`
	Path     string  `json:"path"`
}

// ytdlpOutput captures the relevant fields from yt-dlp --print-json
type ytdlpOutput struct {
	Title    string  `json:"title"`
	Filename string  `json:"filename"`
	URL      string  `json:"webpage_url"`
	Duration float64 `json:"duration"`
	Filesize int64   `json:"filesize"`
	Ext      string  `json:"ext"`
}

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "video",
		Aliases: []string{"vid"},
		Short:   "Video download commands (via yt-dlp)",
	}

	cmd.AddCommand(newDownloadCmd())

	return cmd
}

func newDownloadCmd() *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:   "download [url]",
		Short: "Download video from URL via yt-dlp",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDownload(args[0], format)
		},
	}

	cmd.Flags().StringVarP(&format, "format", "f", "", "yt-dlp format string (default: best merged)")

	return cmd
}

func ensureYtdlp() error {
	if _, err := exec.LookPath("yt-dlp"); err == nil {
		return nil
	}

	// Attempt auto-install
	installer, name := findInstaller()
	if installer == nil {
		return output.PrintError("not_installed", "yt-dlp is not installed and no supported package manager found", map[string]any{
			"hint": "Install manually from https://github.com/yt-dlp/yt-dlp",
		})
	}

	fmt.Fprintf(os.Stderr, "yt-dlp not found, installing via %s...\n", name)

	installCmd := exec.Command(installer[0], installer[1:]...)
	installCmd.Stderr = os.Stderr
	if err := installCmd.Run(); err != nil {
		return output.PrintError("install_failed",
			fmt.Sprintf("auto-install via %s failed: %v", name, err),
			map[string]any{"hint": "Install manually: " + installHint()})
	}

	// Verify it's now available
	if _, err := exec.LookPath("yt-dlp"); err != nil {
		return output.PrintError("install_failed",
			"yt-dlp installed but not found in PATH",
			map[string]any{"hint": "You may need to restart your shell"})
	}

	fmt.Fprintf(os.Stderr, "yt-dlp installed successfully.\n")
	return nil
}

func findInstaller() (cmd []string, name string) {
	switch runtime.GOOS {
	case "darwin":
		if _, err := exec.LookPath("brew"); err == nil {
			return []string{"brew", "install", "yt-dlp"}, "brew"
		}
	case "linux":
		if _, err := exec.LookPath("brew"); err == nil {
			return []string{"brew", "install", "yt-dlp"}, "brew"
		}
	}

	// Cross-platform fallback: pipx > pip3 > pip
	for _, pip := range []string{"pipx", "pip3", "pip"} {
		if _, err := exec.LookPath(pip); err == nil {
			if pip == "pipx" {
				return []string{pip, "install", "yt-dlp"}, pip
			}
			return []string{pip, "install", "--user", "yt-dlp"}, pip
		}
	}

	return nil, ""
}

func installHint() string {
	switch runtime.GOOS {
	case "darwin":
		return "brew install yt-dlp"
	default:
		return "pip install yt-dlp"
	}
}

func runDownload(url, format string) error {
	if err := ensureYtdlp(); err != nil {
		return err
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return output.PrintError("download_failed", fmt.Sprintf("cannot determine home directory: %v", err), nil)
	}
	downloadDir := filepath.Join(homeDir, "Downloads")

	cmdArgs := []string{
		"-P", downloadDir,
		"--print-json",
		"-o", "%(title)s.%(ext)s",
	}
	if format != "" {
		cmdArgs = append(cmdArgs, "-f", format)
	}
	cmdArgs = append(cmdArgs, url)

	cmd := exec.Command("yt-dlp", cmdArgs...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return output.PrintError("download_failed",
			fmt.Sprintf("yt-dlp failed: %v", err),
			map[string]any{"stderr": stderr.String()})
	}

	var meta ytdlpOutput
	if err := json.Unmarshal(stdout.Bytes(), &meta); err != nil {
		return output.PrintError("download_failed",
			fmt.Sprintf("failed to parse yt-dlp output: %v", err), nil)
	}

	fullPath := meta.Filename
	if fullPath == "" && meta.Title != "" {
		fullPath = filepath.Join(downloadDir, meta.Title+"."+meta.Ext)
	}
	if !filepath.IsAbs(fullPath) {
		fullPath = filepath.Join(downloadDir, fullPath)
	}

	return output.Print(DownloadResult{
		Title:    meta.Title,
		Filename: filepath.Base(fullPath),
		URL:      meta.URL,
		Duration: meta.Duration,
		Filesize: meta.Filesize,
		Path:     fullPath,
	})
}
