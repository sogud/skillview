//go:build darwin

package updates

import (
	"errors"
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
	if bundle == "" || syscall.Access(filepath.Dir(bundle), 2) != nil {
		// Wails 3.0 stages beside the target for an atomic same-volume rename.
		// The installation parent must still be writable.
		return "Move SkillView to Applications to install updates."
	}
	return ""
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
