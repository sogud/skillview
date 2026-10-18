package library

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

type ArchiveEntry struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	OriginalPath string `json:"originalPath"`
	ArchivedPath string `json:"archivedPath"`
	ArchivedAt   string `json:"archivedAt"`
	CanRestore   bool   `json:"canRestore"`
	Warning      string `json:"warning"`
}

type ArchiveCatalog struct {
	Directory string         `json:"directory"`
	Entries   []ArchiveEntry `json:"entries"`
	Issues    []string       `json:"issues"`
}

type ArchivePreview struct {
	TrashPreview
	Directory string `json:"directory"`
}

// Each item carries its recovery record, independently of the disposable app cache.
type archiveRecord struct {
	ArchiveEntry
	Version          int    `json:"version"`
	State            string `json:"state"`
	Stage            string `json:"stage"`
	ParentIdentity   string `json:"parentIdentity"`
	OriginalIdentity string `json:"originalIdentity"`
	Identity         string `json:"identity"`
}

// Used only by isolated native acceptance. Ordinary launches use the shared Agent folder.
func (l *Library) SetArchiveDirectory(path string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.archiveDir = path
}

func (l *Library) archiveDirectory() (string, error) {
	if l.archiveDir != "" {
		if !filepath.IsAbs(l.archiveDir) {
			return "", fmt.Errorf("Archive directory must be absolute.")
		}
		parent, err := filepath.EvalSymlinks(filepath.Dir(l.archiveDir))
		if err != nil {
			return "", err
		}
		return filepath.Join(parent, filepath.Base(l.archiveDir)), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	parent := filepath.Join(home, ".agents")
	if resolved, err := filepath.EvalSymlinks(parent); err == nil {
		parent = resolved
	}
	return filepath.Join(parent, ".skillview-archive"), nil
}

func (l *Library) PlanArchive(id string) (ArchivePreview, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.planArchive(id)
}

func (l *Library) planArchive(id string) (ArchivePreview, error) {
	preview, err := l.planTrash(id, "")
	if err != nil {
		return ArchivePreview{}, err
	}
	directory, err := l.archiveDirectory()
	if err != nil {
		return ArchivePreview{}, err
	}
	if _, inside := within(preview.Path, directory); inside {
		return ArchivePreview{}, fmt.Errorf("The archive must be outside this Skill.")
	}
	return ArchivePreview{TrashPreview: preview, Directory: directory}, nil
}

func writeArchiveRecord(directory *os.File, record archiveRecord) error {
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	name, err := randomName(".receipt-")
	if err != nil {
		return err
	}
	fd, err := unix.Openat(int(directory.Fd()), name, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0600)
	if err != nil {
		return err
	}
	file := os.NewFile(uintptr(fd), name)
	defer file.Close()
	defer unix.Unlinkat(int(directory.Fd()), name, 0)
	if _, err = file.Write(data); err != nil {
		return err
	}
	if err = file.Sync(); err != nil {
		return err
	}
	if err = unix.Renameat(int(directory.Fd()), name, int(directory.Fd()), "receipt.json"); err != nil {
		return err
	}
	return directory.Sync()
}

func (l *Library) moveArchiveItem(source *os.File, name, sourcePath string, target *os.File, targetName, targetPath string) error {
	if l.archiveMover != nil {
		return l.archiveMover(sourcePath, targetPath)
	}
	err := unix.RenameatxNp(int(source.Fd()), name, int(target.Fd()), targetName, unix.RENAME_EXCL)
	if !errors.Is(err, unix.EXDEV) {
		return err
	}
	// Foundation preserves file metadata and symbolic links during cross-volume moves.
	// Its path API is used only after rechecking both held parent directories.
	for _, pair := range []struct {
		file *os.File
		path string
	}{{source, sourcePath}, {target, targetPath}} {
		actual, pathErr := pathIdentity(filepath.Dir(pair.path))
		held, heldErr := descriptorIdentity(pair.file)
		if pathErr != nil || heldErr != nil || actual != held {
			return fmt.Errorf("Archive folder changed. Try again.")
		}
	}
	return macArchiveMove(sourcePath, targetPath)
}

func (l *Library) ArchiveSkill(id, token string) (ArchiveEntry, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	preview, err := l.planArchive(id)
	if err != nil {
		return ArchiveEntry{}, err
	}
	if token == "" || token != preview.Token {
		return ArchiveEntry{}, fmt.Errorf("Files changed. Review the archive action again.")
	}
	if err = os.MkdirAll(preview.Directory, 0700); err != nil {
		return ArchiveEntry{}, err
	}
	root, err := openAbsoluteDirectory(preview.Directory)
	if err != nil {
		return ArchiveEntry{}, err
	}
	defer root.Close()
	parent, err := openAbsoluteDirectory(filepath.Dir(preview.Path))
	if err != nil {
		return ArchiveEntry{}, err
	}
	defer parent.Close()
	record := archiveRecord{Version: 1, State: "pending", OriginalIdentity: preview.identity}
	record.ID, err = randomName("archive-")
	if err != nil {
		return ArchiveEntry{}, err
	}
	record.Stage, err = randomName(".skillview-archiving-")
	if err != nil {
		return ArchiveEntry{}, err
	}
	record.ParentIdentity, err = descriptorIdentity(parent)
	if err != nil {
		return ArchiveEntry{}, err
	}
	record.Name = filepath.Base(preview.Path)
	for _, skill := range l.catalog.Skills {
		if skill.ID == id {
			record.Name = skill.Name
			break
		}
	}
	record.OriginalPath = preview.Path
	record.ArchivedAt = time.Now().UTC().Format(time.RFC3339)
	name := filepath.Base(preview.Path)
	record.ArchivedPath = filepath.Join(preview.Directory, record.ID, name)
	if err = unix.Mkdirat(int(root.Fd()), record.ID, 0700); err != nil {
		return ArchiveEntry{}, err
	}
	entry, err := openChild(root, record.ID, true)
	if err != nil {
		return ArchiveEntry{}, err
	}
	defer entry.Close()
	// Persist the original and staging locations before changing the Skill's name.
	if err = writeArchiveRecord(entry, record); err != nil {
		return ArchiveEntry{}, err
	}
	if err = unix.RenameatxNp(int(parent.Fd()), name, int(parent.Fd()), record.Stage, unix.RENAME_EXCL); err != nil {
		return ArchiveEntry{}, err
	}
	stagePath := filepath.Join(filepath.Dir(preview.Path), record.Stage)
	stat, statErr := statAt(parent, record.Stage)
	if statErr != nil || fmt.Sprintf("%d:%d", stat.Dev, stat.Ino) != record.OriginalIdentity {
		_ = unix.RenameatxNp(int(parent.Fd()), record.Stage, int(parent.Fd()), name, unix.RENAME_EXCL)
		return ArchiveEntry{}, fmt.Errorf("Files changed. Review the archive action again.")
	}
	if err = l.moveArchiveItem(parent, record.Stage, stagePath, entry, name, record.ArchivedPath); err != nil {
		if rollback := unix.RenameatxNp(int(parent.Fd()), record.Stage, int(parent.Fd()), name, unix.RENAME_EXCL); rollback == nil {
			record.State = "failed"
			_ = writeArchiveRecord(entry, record)
			return ArchiveEntry{}, fmt.Errorf("Archiving failed. The original Skill was kept.")
		}
		return ArchiveEntry{}, fmt.Errorf("Archiving was interrupted. Open archived Skills to recover the saved files.")
	}
	_ = parent.Sync()
	_ = entry.Sync()
	stat, err = statAt(entry, name)
	if err != nil {
		return ArchiveEntry{}, err
	}
	record.Identity = fmt.Sprintf("%d:%d", stat.Dev, stat.Ino)
	record.State = "archived"
	record.CanRestore = true
	if err = writeArchiveRecord(entry, record); err != nil {
		record.Warning = "Skill archived. The recovery record needs attention."
	}
	return record.ArchiveEntry, nil
}

func readArchiveRecord(root *os.File, directory, id string) (archiveRecord, *os.File, error) {
	if !strings.HasPrefix(id, "archive-") || validateRelative(id) != nil || filepath.Base(id) != id {
		return archiveRecord{}, nil, fmt.Errorf("Invalid archive entry.")
	}
	entry, err := openChild(root, id, true)
	if err != nil {
		return archiveRecord{}, nil, err
	}
	file, err := openChild(entry, "receipt.json", false)
	if err != nil {
		entry.Close()
		return archiveRecord{}, nil, err
	}
	data, err := readRegularText(file)
	file.Close()
	var record archiveRecord
	if err == nil {
		err = json.Unmarshal(data, &record)
	}
	if err == nil && (record.Version != 1 || record.ID != id || !filepath.IsAbs(record.OriginalPath) || filepath.Clean(record.OriginalPath) != record.OriginalPath || validateRelative(filepath.Base(record.OriginalPath)) != nil || record.ParentIdentity == "" || record.OriginalIdentity == "" || !strings.HasPrefix(record.Stage, ".skillview-archiving-") || filepath.Base(record.Stage) != record.Stage || record.ArchivedPath != filepath.Join(directory, id, filepath.Base(record.OriginalPath))) {
		err = fmt.Errorf("Invalid archive entry.")
	}
	if err != nil {
		entry.Close()
		return archiveRecord{}, nil, err
	}
	return record, entry, nil
}

// A pending record can recover either a completed move or its original staged item.
func archiveRecoveryItem(record archiveRecord, entry *os.File) (string, string, error) {
	if record.State == "failed" {
		return "", "", fmt.Errorf("An incomplete archive was kept. Inspect the archive folder.")
	}
	if record.State == "pending" {
		stage := filepath.Join(filepath.Dir(record.OriginalPath), record.Stage)
		parent, openErr := openAbsoluteDirectory(filepath.Dir(stage))
		if openErr == nil {
			defer parent.Close()
			parentID, _ := descriptorIdentity(parent)
			staged, stageErr := statAt(parent, record.Stage)
			if stageErr == nil && parentID == record.ParentIdentity && fmt.Sprintf("%d:%d", staged.Dev, staged.Ino) == record.OriginalIdentity {
				return stage, record.OriginalIdentity, nil
			}
		}
		if _, originalErr := os.Lstat(record.OriginalPath); originalErr == nil {
			return "", "", fmt.Errorf("An incomplete archive was kept. Inspect the archive folder.")
		}
	}
	name := filepath.Base(record.OriginalPath)
	stat, err := statAt(entry, name)
	if err == nil && stat.Mode&unix.S_IFMT == unix.S_IFDIR {
		identity := fmt.Sprintf("%d:%d", stat.Dev, stat.Ino)
		if record.Identity == "" || identity == record.Identity {
			return record.ArchivedPath, identity, nil
		}
	}
	return "", "", fmt.Errorf("Archived item changed or is missing. Inspect the archive folder.")
}

func (l *Library) ArchivedSkills() (ArchiveCatalog, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	directory, err := l.archiveDirectory()
	result := ArchiveCatalog{Directory: directory, Entries: []ArchiveEntry{}, Issues: []string{}}
	if err != nil {
		return result, err
	}
	root, err := openAbsoluteDirectory(directory)
	if errors.Is(err, os.ErrNotExist) {
		return result, nil
	}
	if err != nil {
		return result, err
	}
	defer root.Close()
	entries, err := root.ReadDir(-1)
	if err != nil {
		return result, err
	}
	for _, child := range entries {
		if !child.IsDir() || !strings.HasPrefix(child.Name(), "archive-") {
			continue
		}
		record, entry, err := readArchiveRecord(root, directory, child.Name())
		if err != nil {
			result.Issues = append(result.Issues, "An archive record could not be read. Inspect the archive folder.")
			continue
		}
		path, _, recoveryErr := archiveRecoveryItem(record, entry)
		_, payloadErr := statAt(entry, filepath.Base(record.OriginalPath))
		entry.Close()
		// An unstarted move or a restore completed just before a crash leaves only metadata.
		if recoveryErr != nil && errors.Is(payloadErr, os.ErrNotExist) {
			if _, originalErr := os.Lstat(record.OriginalPath); originalErr == nil {
				continue
			}
		}
		if record.State == "failed" {
			if payloadErr == nil {
				result.Issues = append(result.Issues, recoveryErr.Error())
			}
			continue
		}
		record.CanRestore = recoveryErr == nil
		if recoveryErr != nil {
			record.Warning = recoveryErr.Error()
		} else if path != record.ArchivedPath {
			record.Warning = "Archiving was interrupted. Restore this item to its original location."
		}
		result.Entries = append(result.Entries, record.ArchiveEntry)
	}
	sort.Slice(result.Entries, func(i, j int) bool { return result.Entries[i].ArchivedAt > result.Entries[j].ArchivedAt })
	return result, nil
}

func (l *Library) RestoreArchivedSkill(id string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	directory, err := l.archiveDirectory()
	if err != nil {
		return err
	}
	root, err := openAbsoluteDirectory(directory)
	if err != nil {
		return err
	}
	defer root.Close()
	record, entry, err := readArchiveRecord(root, directory, id)
	if err != nil {
		return err
	}
	defer entry.Close()
	path, identity, err := archiveRecoveryItem(record, entry)
	if err != nil {
		return err
	}
	parent, err := openAbsoluteDirectory(filepath.Dir(record.OriginalPath))
	if err != nil {
		return fmt.Errorf("The original folder is unavailable. Inspect the archive folder.")
	}
	defer parent.Close()
	parentID, _ := descriptorIdentity(parent)
	if parentID != record.ParentIdentity {
		return fmt.Errorf("The original folder changed. Inspect the archive folder.")
	}
	name := filepath.Base(record.OriginalPath)
	if _, err = statAt(parent, name); !errors.Is(err, unix.ENOENT) {
		return fmt.Errorf("The original location is occupied. Both copies were kept.")
	}
	source := entry
	if path != record.ArchivedPath {
		source = parent
	}
	sourceName := filepath.Base(path)
	stat, err := statAt(source, sourceName)
	if err != nil || fmt.Sprintf("%d:%d", stat.Dev, stat.Ino) != identity {
		return fmt.Errorf("Archive folder changed. Try again.")
	}
	if path == record.ArchivedPath {
		record.State = "restoring"
		record.Identity = identity
	}
	if err = writeArchiveRecord(entry, record); err != nil {
		return err
	}
	if err = l.moveArchiveItem(source, sourceName, path, parent, name, record.OriginalPath); err != nil {
		return fmt.Errorf("Restoring failed. Inspect the archive folder; saved files were kept.")
	}
	_ = parent.Sync()
	_ = source.Sync()
	if path != record.ArchivedPath {
		record.State = "failed"
		_ = writeArchiveRecord(entry, record)
	}
	// Remove only our recovery metadata, never unexpected files or partial copies.
	if _, err := statAt(entry, filepath.Base(record.ArchivedPath)); errors.Is(err, unix.ENOENT) {
		_ = unix.Unlinkat(int(entry.Fd()), "receipt.json", 0)
		_ = unix.Unlinkat(int(root.Fd()), id, unix.AT_REMOVEDIR)
	}
	return nil
}
