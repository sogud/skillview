//go:build darwin

package updates

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

func installReason(fixture bool) string {
	if fixture {
		return "Updates cannot be installed in a preview window."
	}
	bundle := appBundle()
	if bundle == "" || !sameDevice(filepath.Dir(bundle), os.TempDir()) || syscall.Access(filepath.Dir(bundle), 2) != nil {
		// The pinned Wails helper renames its staged bundle. Cross-volume
		// moves and read-only installation directories need manual installation.
		return "Move SkillView to Applications to install updates."
	}
	return ""
}

func sameDevice(first, second string) bool {
	var a, b syscall.Stat_t
	return syscall.Stat(first, &a) == nil && syscall.Stat(second, &b) == nil && a.Dev == b.Dev
}

func validateBundle(path, version string) error {
	if filepath.Ext(path) != ".app" {
		return errors.New("The update does not contain a SkillView app.")
	}
	plist := filepath.Join(path, "Contents", "Info.plist")
	for key, expected := range map[string]string{"CFBundleIdentifier": "me.sogud.skillview", "SkillViewReleaseVersion": version} {
		value, err := exec.Command("/usr/libexec/PlistBuddy", "-c", "Print "+key, plist).Output()
		if err != nil || strings.TrimSpace(string(value)) != expected {
			return errors.New("The downloaded app does not match this release.")
		}
	}
	if err := exec.Command("/usr/bin/codesign", "--verify", "--strict", path).Run(); err != nil {
		return errors.New("The downloaded app signature is invalid.")
	}
	return nil
}
