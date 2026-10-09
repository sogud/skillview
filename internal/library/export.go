package library

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"golang.org/x/sys/unix"
)

// Export writes a separate new file. Neither originals nor existing exports
// can be replaced, including paths reached through links.
func (l *Library) Export(id, path, revision, destination, text string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(text) > maxTextBytes || !utf8.ValidString(text) || strings.ContainsRune(text, 0) {
		return fmt.Errorf("save requires UTF-8 text smaller than 2 MiB")
	}
	doc, err := l.read(id, path)
	if err != nil {
		return err
	}
	if revision == "" || doc.Revision != revision {
		return fmt.Errorf("This file changed. Reopen it before translating.")
	}
	if doc.Kind != "markdown" && doc.Kind != "text" {
		return fmt.Errorf("only text documents can be translated")
	}
	if !filepath.IsAbs(destination) || strings.ContainsRune(destination, 0) {
		return fmt.Errorf("choose a new export location")
	}
	destination = filepath.Clean(destination)
	selectedParent := filepath.Dir(destination)
	parentPath, err := filepath.EvalSymlinks(selectedParent)
	if err != nil {
		return err
	}
	destination = filepath.Join(parentPath, filepath.Base(destination))
	for _, skill := range l.catalog.Skills {
		root := skill.Path
		if l.refs[skill.ID].fileName != "" {
			root = filepath.Dir(root)
		}
		if _, inside := within(root, destination); inside {
			return fmt.Errorf("choose an export location outside the original Skill folders")
		}
	}
	parent, err := openAbsoluteDirectory(parentPath)
	if err != nil {
		return err
	}
	defer parent.Close()
	identity, err := descriptorIdentity(parent)
	if err != nil {
		return err
	}
	temporary, err := randomName(".skillview-export-")
	if err != nil {
		return err
	}
	fd, err := unix.Openat(int(parent.Fd()), temporary, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0600)
	if err != nil {
		return err
	}
	file := os.NewFile(uintptr(fd), temporary)
	defer file.Close()
	defer unix.Unlinkat(int(parent.Fd()), temporary, 0)
	if _, err = file.WriteString(text); err != nil {
		return err
	}
	if err = file.Sync(); err != nil {
		return err
	}
	current, err := l.read(id, path)
	if err != nil {
		return err
	}
	if current.Revision != revision {
		return fmt.Errorf("This file changed. Reopen it before translating.")
	}
	currentParent, err := filepath.EvalSymlinks(selectedParent)
	if err != nil || currentParent != parentPath {
		return fmt.Errorf("export folder changed; choose it again")
	}
	checked, err := openAbsoluteDirectory(currentParent)
	if err != nil {
		return err
	}
	defer checked.Close()
	checkedIdentity, err := descriptorIdentity(checked)
	if err != nil || identity != checkedIdentity {
		return fmt.Errorf("export folder changed; choose it again")
	}
	if err = unix.RenameatxNp(int(parent.Fd()), temporary, int(checked.Fd()), filepath.Base(destination), unix.RENAME_EXCL); err != nil {
		return fmt.Errorf("export could not be created; choose a new filename. Existing files were kept")
	}
	return checked.Sync()
}
