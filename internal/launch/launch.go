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
	root, err := Bundle(executable)
	if err != nil {
		return err
	}
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
	args = append(args, root)
	return exec.Command("/usr/bin/open", args...).Run()
}

func InstallCommand(executable, home string) (string, error) {
	if _, err := Bundle(executable); err != nil {
		return "", err
	}
	executable, err := filepath.EvalSymlinks(executable)
	if err != nil {
		return "", err
	}
	directory := filepath.Join(home, ".local", "bin")
	if err = os.MkdirAll(directory, 0755); err != nil {
		return "", err
	}
	target := filepath.Join(directory, "skillview")
	if existing, err := os.Readlink(target); err == nil && existing == executable {
		return target, nil
	}
	if err = os.Symlink(executable, target); err != nil {
		return "", fmt.Errorf("Command already exists or cannot be created: %s", target)
	}
	return target, nil
}
