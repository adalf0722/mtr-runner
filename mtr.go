package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// extraPaths covers common locations that may not be in PATH when
// the binary is launched from a browser or non-login shell.
var extraPaths = []string{
	"/usr/local/sbin",
	"/usr/local/bin",
	"/opt/homebrew/sbin",
	"/opt/homebrew/bin",
}

func runMtr(target string) (string, error) {
	searchPaths := append(filepath.SplitList(os.Getenv("PATH")), extraPaths...)
	mtrPath, err := findMtrBinary(searchPaths)
	if err != nil {
		return "", fmt.Errorf("找不到 mtr：%w\n請先安裝：%s", err, installHint())
	}

	args := buildMtrArgs(target)
	cmd := exec.Command(mtrPath, args...)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("mtr 執行錯誤：%w\n%s", err, string(exitErr.Stderr))
		}
		return "", fmt.Errorf("mtr 執行錯誤：%w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func findMtrBinary(searchPaths []string) (string, error) {
	names := []string{"mtr"}
	if runtime.GOOS == "windows" {
		names = []string{"mtr.exe", "winmtr.exe"}
	}
	for _, dir := range searchPaths {
		for _, name := range names {
			p := filepath.Join(dir, name)
			if info, err := os.Stat(p); err == nil && !info.IsDir() {
				return p, nil
			}
		}
	}
	return "", errors.New("mtr binary not found in PATH")
}

func buildMtrArgs(target string) []string {
	return []string{"--json", "--report", "--report-cycles", "10", target}
}

func installHint() string {
	switch runtime.GOOS {
	case "darwin":
		return "brew install mtr"
	case "linux":
		return "sudo apt install mtr-tiny  # 或 sudo yum install mtr"
	default:
		return "請至 https://www.bitwizard.nl/mtr/ 下載 WinMTR"
	}
}
