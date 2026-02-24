//go:build windows

package platform

import "os/exec"

func OpenBrowser(url string) error {
	return exec.Command("cmd", "/c", "start", url).Start()
}
