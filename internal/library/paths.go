package library

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
)

func validateRelative(path string) error {
	if path == "" || filepath.IsAbs(path) || strings.ContainsRune(path, 0) {
		return fmt.Errorf("choose a relative file inside this Skill")
	}
	for _, part := range strings.Split(filepath.ToSlash(path), "/") {
		if part == "" || part == "." || part == ".." {
			return fmt.Errorf("parent paths and empty segments are not allowed")
		}
	}
	return nil
}

func descriptorIdentity(file *os.File) (string, error) {
	var stat unix.Stat_t
	if err := unix.Fstat(int(file.Fd()), &stat); err != nil {
		return "", err
	}
	return fmt.Sprintf("%d:%d", stat.Dev, stat.Ino), nil
}

func pathIdentity(path string) (string, error) {
	file, err := openAbsoluteDirectory(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	return descriptorIdentity(file)
}

// Open each component through the preceding directory. A swapped symlink
// cannot redirect the operation to a different root or resource.
func openAbsoluteDirectory(path string) (*os.File, error) {
	if !filepath.IsAbs(path) {
		return nil, fmt.Errorf("source root must be absolute")
	}
	fd, err := unix.Open("/", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	current := os.NewFile(uintptr(fd), "/")
	parts := strings.Split(strings.TrimPrefix(filepath.Clean(path), "/"), "/")
	for index, part := range parts {
		if part == "" {
			continue
		}
		// Intermediate directories need traversal permission, not listing
		// permission. macOS volume Trash parents commonly have mode 01311.
		flags := directorySearchFlag | unix.O_DIRECTORY | unix.O_NOFOLLOW | unix.O_CLOEXEC
		if index == len(parts)-1 {
			flags = unix.O_RDONLY | unix.O_DIRECTORY | unix.O_NOFOLLOW | unix.O_CLOEXEC
		}
		childFD, err := unix.Openat(int(current.Fd()), part, flags, 0)
		current.Close()
		if err != nil {
			return nil, fmt.Errorf("directory changed or contains a link; select it again: %w", err)
		}
		current = os.NewFile(uintptr(childFD), part)
	}
	return current, nil
}

func openChild(parent *os.File, name string, directory bool) (*os.File, error) {
	flags := unix.O_RDONLY | unix.O_NOFOLLOW | unix.O_CLOEXEC | unix.O_NONBLOCK
	if directory {
		flags |= unix.O_DIRECTORY
	}
	fd, err := unix.Openat(int(parent.Fd()), name, flags, 0)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(fd), name), nil
}

func openDirectory(root *os.File, path string) (*os.File, error) {
	if path == "" {
		fd, err := unix.Dup(int(root.Fd()))
		if err != nil {
			return nil, err
		}
		return os.NewFile(uintptr(fd), root.Name()), nil
	}
	if err := validateRelative(path); err != nil {
		return nil, err
	}
	fd, err := unix.Dup(int(root.Fd()))
	if err != nil {
		return nil, err
	}
	current := os.NewFile(uintptr(fd), root.Name())
	for _, part := range strings.Split(filepath.ToSlash(path), "/") {
		next, err := openChild(current, part, true)
		current.Close()
		if err != nil {
			return nil, err
		}
		current = next
	}
	return current, nil
}

func openSource(source Source) (*os.File, error) {
	if !source.Approved {
		return nil, fmt.Errorf("select the original directory to allow reading")
	}
	resolved, err := filepath.EvalSymlinks(source.Path)
	if err != nil {
		return nil, err
	}
	if resolved != source.ResolvedPath {
		return nil, fmt.Errorf("source link changed; select the directory again")
	}
	root, err := openAbsoluteDirectory(source.ResolvedPath)
	if err != nil {
		return nil, err
	}
	identity, err := descriptorIdentity(root)
	if err != nil || identity != source.Identity {
		root.Close()
		return nil, fmt.Errorf("source directory was replaced; select it again")
	}
	return root, nil
}

func within(root, target string) (string, bool) {
	relative, err := filepath.Rel(root, target)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
		return "", false
	}
	if relative == "." {
		relative = ""
	}
	return relative, true
}
