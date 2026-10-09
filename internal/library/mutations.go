package library

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"golang.org/x/sys/unix"
	"gopkg.in/yaml.v3"
)

func randomName(prefix string) (string, error) {
	var bytes [12]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(bytes[:]), nil
}

func (l *Library) Save(id, path, expectedRevision, text string) (SaveResult, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(text) > maxTextBytes || !utf8.ValidString(text) || strings.ContainsRune(text, 0) {
		return SaveResult{}, fmt.Errorf("save requires UTF-8 text smaller than 2 MiB")
	}
	directory, source, err := l.openSkill(id)
	if err != nil {
		return SaveResult{}, err
	}
	defer directory.Close()
	if !source.Writable || !l.skillEditable(id) {
		return SaveResult{}, fmt.Errorf("this source or linked entry is read-only")
	}
	file, parent, name, err := openResource(directory, path)
	if err != nil {
		return SaveResult{}, err
	}
	defer file.Close()
	defer parent.Close()
	var original unix.Stat_t
	if err := unix.Fstat(int(file.Fd()), &original); err != nil {
		return SaveResult{}, err
	}
	if original.Nlink != 1 {
		return SaveResult{}, fmt.Errorf("hard-linked files cannot be replaced")
	}
	data, err := readRegularText(file)
	if err != nil {
		return SaveResult{}, err
	}
	if expectedRevision == "" || digest(data) != expectedRevision {
		doc, err := l.read(id, path)
		return SaveResult{Status: "conflict", Document: doc}, err
	}
	if string(data) == text {
		doc, err := l.read(id, path)
		return SaveResult{Status: "saved", Document: doc}, err
	}
	recoveryDir := filepath.Join(l.dataDir, "Recovery")
	if err := os.MkdirAll(recoveryDir, 0700); err != nil {
		return SaveResult{}, err
	}
	recovery, err := os.CreateTemp(recoveryDir, "original-*.txt")
	if err != nil {
		return SaveResult{}, err
	}
	recoveryPath := recovery.Name()
	_, err = recovery.Write(data)
	if err == nil {
		err = recovery.Sync()
	}
	closeErr := recovery.Close()
	if err != nil {
		return SaveResult{}, err
	}
	if closeErr != nil {
		return SaveResult{}, closeErr
	}
	tempName, err := randomName(".skillview-save-")
	if err != nil {
		return SaveResult{}, err
	}
	fd, err := unix.Openat(int(parent.Fd()), tempName, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0600)
	if err != nil {
		return SaveResult{}, err
	}
	temp := os.NewFile(uintptr(fd), tempName)
	defer temp.Close()
	defer unix.Unlinkat(int(parent.Fd()), tempName, 0)
	if _, err := io.WriteString(temp, text); err != nil {
		return SaveResult{}, err
	}
	if err := copyMetadata(file, temp); err != nil {
		return SaveResult{}, fmt.Errorf("file metadata could not be preserved; original kept: %w", err)
	}
	if err := unix.Fchown(fd, int(original.Uid), int(original.Gid)); err != nil {
		return SaveResult{}, err
	}
	if err := unix.Fchmod(fd, uint32(original.Mode&07777)); err != nil {
		return SaveResult{}, err
	}
	if err := temp.Sync(); err != nil {
		return SaveResult{}, err
	}
	// Verify the selected root and every parent again immediately before commit.
	checkDir, _, err := l.openSkill(id)
	if err != nil {
		return SaveResult{}, err
	}
	checkFile, checkParent, _, err := openResource(checkDir, path)
	checkDir.Close()
	if err != nil {
		return SaveResult{}, err
	}
	defer checkFile.Close()
	defer checkParent.Close()
	parentID, err := descriptorIdentity(parent)
	if err != nil {
		return SaveResult{}, err
	}
	checkParentID, err := descriptorIdentity(checkParent)
	if err != nil {
		return SaveResult{}, err
	}
	current, err := statAt(checkParent, name)
	if err != nil {
		return SaveResult{}, err
	}
	if parentID != checkParentID || current.Dev != original.Dev || current.Ino != original.Ino || current.Nlink != 1 {
		return SaveResult{}, fmt.Errorf("file or parent was replaced; draft kept, original backed up at %s", recoveryPath)
	}
	currentData, err := readRegularText(checkFile)
	if err != nil {
		return SaveResult{}, err
	}
	if digest(currentData) != expectedRevision {
		doc, err := l.read(id, path)
		return SaveResult{Status: "conflict", Document: doc, RecoveryPath: recoveryPath}, err
	}
	if err := unix.Renameat(int(parent.Fd()), tempName, int(parent.Fd()), name); err != nil {
		return SaveResult{}, err
	}
	if err := parent.Sync(); err != nil {
		return SaveResult{}, fmt.Errorf("file was saved but directory sync failed; refresh before saving again: %w", err)
	}
	doc, err := l.read(id, path)
	return SaveResult{Status: "saved", Document: doc, RecoveryPath: recoveryPath}, err
}

func (l *Library) Create(sourceID, name, description string) (string, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !validName.MatchString(name) || len(name) > 64 {
		return "", fmt.Errorf("name must use lowercase letters, numbers, and single hyphens (up to 64 characters)")
	}
	if strings.TrimSpace(description) == "" || utf8.RuneCountInString(description) > 1024 {
		return "", fmt.Errorf("add a description of up to 1024 characters")
	}
	source, err := l.source(sourceID)
	if err != nil {
		return "", err
	}
	if !source.Writable {
		return "", fmt.Errorf("select a writable source")
	}
	root, err := openSource(source)
	if err != nil {
		return "", err
	}
	defer root.Close()
	if existing, err := openChild(root, "SKILL.md", false); err == nil {
		existing.Close()
		return "", fmt.Errorf("this source is a single Skill; choose its containing Skills folder to create a new Skill")
	}
	var metadata struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
	}
	metadata.Name = name
	metadata.Description = description
	header, err := yaml.Marshal(metadata)
	if err != nil {
		return "", err
	}
	contents := "---\n" + string(header) + "---\n\n# " + name + "\n\n"
	tempName, err := randomName(".skillview-create-")
	if err != nil {
		return "", err
	}
	if err := unix.Mkdirat(int(root.Fd()), tempName, 0755); err != nil {
		return "", err
	}
	defer unix.Unlinkat(int(root.Fd()), tempName, unix.AT_REMOVEDIR)
	tempDir, err := openChild(root, tempName, true)
	if err != nil {
		return "", err
	}
	defer tempDir.Close()
	fd, err := unix.Openat(int(tempDir.Fd()), "SKILL.md", unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0644)
	if err != nil {
		return "", err
	}
	file := os.NewFile(uintptr(fd), "SKILL.md")
	defer file.Close()
	published := false
	defer func() {
		if !published {
			_ = unix.Unlinkat(int(tempDir.Fd()), "SKILL.md", 0)
		}
	}()
	if _, err := io.WriteString(file, contents); err != nil {
		return "", err
	}
	if err := file.Sync(); err != nil {
		return "", err
	}
	if err := tempDir.Sync(); err != nil {
		return "", err
	}
	check, err := openSource(source)
	if err != nil {
		return "", err
	}
	check.Close()
	if err := unix.RenameatxNp(int(root.Fd()), tempName, int(root.Fd()), name, unix.RENAME_EXCL); err != nil {
		return "", fmt.Errorf("could not create %s; an existing directory is never replaced: %w", name, err)
	}
	published = true
	// Do not remove the file after publishing the directory.
	// Deferred cleanup only removes the unpublished temporary path.
	if err := root.Sync(); err != nil {
		return "", fmt.Errorf("Skill created but directory sync failed; rescan: %w", err)
	}
	return objectID(filepath.Join(source.ResolvedPath, name)), nil
}
