package cli

import (
	"fmt"
	"os/exec"
	"runtime"
)

func (a *app) clipboard() (string, error) {
	if a.deps.Clipboard != nil {
		return a.deps.Clipboard()
	}
	var candidates [][]string
	switch runtime.GOOS {
	case "darwin":
		candidates = [][]string{{"pbpaste"}}
	case "linux":
		candidates = [][]string{{"wl-paste", "--no-newline"}, {"xclip", "-selection", "clipboard", "-o"}, {"xsel", "--clipboard", "--output"}}
	default:
		return "", fmt.Errorf("%s ではクリップボード読み取りに対応していません。cURL を標準入力へ渡してください", runtime.GOOS)
	}
	var lastErr error
	for _, candidate := range candidates {
		if _, err := exec.LookPath(candidate[0]); err != nil {
			lastErr = err
			continue
		}
		data, err := exec.Command(candidate[0], candidate[1:]...).Output()
		if err == nil {
			return string(data), nil
		}
		lastErr = err
	}
	return "", fmt.Errorf("クリップボードを読み取れません: %w", lastErr)
}
