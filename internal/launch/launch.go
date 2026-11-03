package launch

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const BundleID = "me.sogud.skillview"

func DataDirectory() (string, error) {
	if path := os.Getenv("SKILLVIEW_DATA_DIR"); path != "" {
		return filepath.Abs(path)
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "SkillView"), nil
}

func Bundle(executable string) (string, error) {
	path, err := filepath.EvalSymlinks(executable)
	if err != nil {
		return "", err
	}
	if filepath.Base(filepath.Dir(path)) != "MacOS" || filepath.Base(filepath.Dir(filepath.Dir(path))) != "Contents" {
		return "", fmt.Errorf("Use a packaged SkillView app.")
	}
	root := filepath.Dir(filepath.Dir(filepath.Dir(path)))
	if filepath.Ext(root) != ".app" {
		return "", fmt.Errorf("Use a packaged SkillView app.")
	}
	return root, nil
}

func OpenApp(executable string) error {
	root, bundleErr := Bundle(executable)
	dataDir, err := DataDirectory()
	if err != nil {
		return err
	}
	// A second launch reaches the running app even when it is already focused.
	args := []string{"-n", "--env", "SKILLVIEW_DATA_DIR=" + dataDir}
	if os.Getenv("SKILLVIEW_SOURCE") != "" {
		for _, name := range []string{"HOME", "SKILLVIEW_SOURCE", "SKILLVIEW_FIXTURE_EDIT", "SKILLVIEW_PLUGIN_FIXTURE"} {
			args = append(args, "--env", name+"="+os.Getenv(name))
		}
	}
	if bundleErr == nil {
		args = append(args, root)
	} else {
		args = append(args, "-b", BundleID)
	}
	if err := exec.Command("/usr/bin/open", args...).Run(); err != nil {
		return fmt.Errorf("SkillView app could not be opened. Install the macOS app to use 'skillview app'; other CLI commands do not need it: %w", err)
	}
	return nil
}

func InstallCommand(executable, home string) (string, error) {
	executable, err := filepath.EvalSymlinks(executable)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(executable)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0111 == 0 {
		return "", fmt.Errorf("CLI executable is not an executable file: %s", executable)
	}
	directory := filepath.Join(home, ".local", "bin")
	if err = os.MkdirAll(directory, 0755); err != nil {
		return "", err
	}
	target := filepath.Join(directory, "skillview")
	if existing, err := os.Stat(target); err == nil && os.SameFile(info, existing) {
		return target, nil
	}
	if err = os.Symlink(executable, target); err != nil {
		return "", fmt.Errorf("Command already exists or cannot be created: %s", target)
	}
	return target, nil
}
