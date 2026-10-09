package library

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
	"golang.org/x/sys/unix"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

type TrashPreview struct {
	Path       string   `json:"path"`
	Count      int      `json:"count"`
	Token      string   `json:"token"`
	Links      []string `json:"links"`
	Link       bool     `json:"link"`
	LinkTarget string   `json:"linkTarget"`
	identity   string
}

func resourceRevision(file *os.File) string {
	var stat unix.Stat_t
	if unix.Fstat(int(file.Fd()), &stat) != nil {
		return ""
	}
	return digest([]byte(fmt.Sprintf("%d:%d:%d:%d:%d:%d:%d:%d", stat.Dev, stat.Ino, stat.Size, stat.Mtim.Sec, stat.Mtim.Nsec, stat.Ctim.Sec, stat.Ctim.Nsec, stat.Nlink)))
}
func resourcePath(path string) error {
	if err := validateRelative(path); err != nil {
		return err
	}
	if strings.EqualFold(filepath.Base(path), "SKILL.md") || strings.HasPrefix(filepath.Base(path), ".skillview-") {
		return fmt.Errorf("SKILL.md is managed as the main document")
	}
	return nil
}
func (l *Library) writableSkill(id string) (*os.File, Source, error) {
	dir, source, err := l.openSkill(id)
	if err != nil {
		return nil, Source{}, err
	}
	if !source.Writable || !l.skillEditable(id) || l.refs[id].fileName != "" {
		dir.Close()
		return nil, Source{}, fmt.Errorf("this source or linked entry is read-only")
	}
	return dir, source, nil
}
func (l *Library) CreateResource(id, path, text string) (Document, error) {
	if len(text) > maxTextBytes || !utf8.ValidString(text) || strings.ContainsRune(text, 0) {
		return Document{}, fmt.Errorf("save requires UTF-8 text smaller than 2 MiB")
	}
	if err := l.ImportResource(id, path, []byte(text)); err != nil {
		return Document{}, err
	}
	return l.Read(id, path)
}
func (l *Library) ImportResource(id, path string, data []byte) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := resourcePath(path); err != nil {
		return err
	}
	if len(data) > 16*1024*1024 {
		return fmt.Errorf("choose a resource smaller than 16 MiB")
	}
	dir, _, err := l.writableSkill(id)
	if err != nil {
		return err
	}
	defer dir.Close()
	parentPath := filepath.Dir(path)
	if parentPath == "." {
		parentPath = ""
	}
	parent, err := openDirectory(dir, parentPath)
	if err != nil {
		return fmt.Errorf("choose an existing resource folder")
	}
	defer parent.Close()
	// Write privately, then publish with an exclusive rename. Existing files survive.
	temporary, err := randomName(".skillview-resource-")
	if err != nil {
		return err
	}
	fd, err := unix.Openat(int(parent.Fd()), temporary, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0644)
	if err != nil {
		return err
	}
	file := os.NewFile(uintptr(fd), temporary)
	defer file.Close()
	defer unix.Unlinkat(int(parent.Fd()), temporary, 0)
	if _, err = file.Write(data); err != nil {
		return err
	}
	if err = file.Sync(); err != nil {
		return err
	}
	check, _, err := l.writableSkill(id)
	if err != nil {
		return err
	}
	defer check.Close()
	checkParent, err := openDirectory(check, parentPath)
	if err != nil {
		return err
	}
	defer checkParent.Close()
	a, _ := descriptorIdentity(parent)
	b, _ := descriptorIdentity(checkParent)
	if a != b {
		return fmt.Errorf("resource folder changed; refresh before trying again")
	}
	if err = unix.RenameatxNp(int(parent.Fd()), temporary, int(checkParent.Fd()), filepath.Base(path), unix.RENAME_EXCL); err != nil {
		return fmt.Errorf("resource could not be created; an existing file was kept: %w", err)
	}
	return checkParent.Sync()
}
func (l *Library) RenameResource(id, path, newPath, expected string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := resourcePath(path); err != nil {
		return err
	}
	if err := resourcePath(newPath); err != nil {
		return err
	}
	dir, _, err := l.writableSkill(id)
	if err != nil {
		return err
	}
	defer dir.Close()
	file, parent, name, err := openResource(dir, path)
	if err != nil {
		return err
	}
	defer file.Close()
	defer parent.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("only ordinary resource files can be renamed")
	}
	links, _ := fileLinks(file)
	if links != 1 {
		return fmt.Errorf("hard-linked files cannot be renamed")
	}
	if expected == "" || resourceRevision(file) != expected {
		return fmt.Errorf("resource changed; refresh before trying again")
	}
	targetPath := filepath.Dir(newPath)
	if targetPath == "." {
		targetPath = ""
	}
	target, err := openDirectory(dir, targetPath)
	if err != nil {
		return fmt.Errorf("choose an existing resource folder")
	}
	defer target.Close()
	check, _, err := l.writableSkill(id)
	if err != nil {
		return err
	}
	defer check.Close()
	current, currentParent, _, err := openResource(check, path)
	if err != nil {
		return err
	}
	defer current.Close()
	defer currentParent.Close()
	originalParent, _ := descriptorIdentity(parent)
	checkedParent, _ := descriptorIdentity(currentParent)
	checkTarget, err := openDirectory(check, targetPath)
	if err != nil {
		return err
	}
	defer checkTarget.Close()
	targetID, _ := descriptorIdentity(target)
	checkedTargetID, _ := descriptorIdentity(checkTarget)
	if originalParent != checkedParent || targetID != checkedTargetID || resourceRevision(current) != expected {
		return fmt.Errorf("resource changed; refresh before trying again")
	}
	if err = unix.RenameatxNp(int(currentParent.Fd()), name, int(checkTarget.Fd()), filepath.Base(newPath), unix.RENAME_EXCL); err != nil {
		return fmt.Errorf("resource could not be renamed; an existing file was kept: %w", err)
	}
	if err = currentParent.Sync(); err != nil {
		return err
	}
	return checkTarget.Sync()
}

func (l *Library) PlanTrash(id, path string) (TrashPreview, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.planTrash(id, path)
}
func (l *Library) planTrash(id, path string) (TrashPreview, error) {
	dir, source, err := l.writableSkill(id)
	if err != nil {
		return TrashPreview{}, err
	}
	defer dir.Close()
	preview := TrashPreview{Path: filepath.Join(source.ResolvedPath, l.refs[id].relativeDir, path), Links: []string{}}
	preview.identity, err = descriptorIdentity(dir)
	if err != nil {
		return TrashPreview{}, err
	}
	if path != "" {
		if err = resourcePath(path); err != nil {
			return TrashPreview{}, err
		}
		parentPath := filepath.Dir(path)
		if parentPath == "." {
			parentPath = ""
		}
		linkParent, err := openDirectory(dir, parentPath)
		if err != nil {
			return TrashPreview{}, err
		}
		stat, statErr := statAt(linkParent, filepath.Base(path))
		if statErr == nil && stat.Mode&unix.S_IFMT == unix.S_IFLNK {
			link, err := inspectLinkAt(linkParent, filepath.Base(path), preview.Path)
			linkParent.Close()
			return link, err
		}
		linkParent.Close()
		file, parent, _, err := openResource(dir, path)
		if err != nil {
			return TrashPreview{}, err
		}
		defer file.Close()
		defer parent.Close()
		info, err := file.Stat()
		if err != nil || !info.Mode().IsRegular() {
			return TrashPreview{}, fmt.Errorf("only ordinary resource files can be moved to Trash")
		}
		links, _ := fileLinks(file)
		if links != 1 {
			return TrashPreview{}, fmt.Errorf("hard-linked files cannot be moved to Trash")
		}
		preview.Count = 1
		preview.Token = resourceRevision(file)
		preview.identity, err = descriptorIdentity(file)
		if err != nil {
			return TrashPreview{}, err
		}
		return preview, nil
	}
	hash := sha256.New()
	count := 0
	var walk func(*os.File, int) error
	walk = func(parent *os.File, depth int) error {
		if depth > 64 || count > 10000 {
			return fmt.Errorf("resource tree is too large; inspect this folder in Finder")
		}
		entries, err := parent.ReadDir(-1)
		if err != nil {
			return err
		}
		sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
		for _, entry := range entries {
			stat, err := statAt(parent, entry.Name())
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintf(hash, "%s:%d:%d:%d:%d:%d:%d;", entry.Name(), stat.Ino, stat.Size, stat.Mtim.Sec, stat.Mtim.Nsec, stat.Ctim.Sec, stat.Ctim.Nsec)
			if entry.IsDir() {
				child, err := openChild(parent, entry.Name(), true)
				if err != nil {
					return err
				}
				err = walk(child, depth+1)
				child.Close()
				if err != nil {
					return err
				}
			} else {
				count++
			}
		}
		return nil
	}
	if err = walk(dir, 0); err != nil {
		return TrashPreview{}, err
	}
	preview.Count = count
	preview.Token = hex.EncodeToString(hash.Sum(nil))
	for _, skill := range l.catalog.Skills {
		if skill.ID == id {
			for _, occurrence := range skill.Occurrences {
				if occurrence.Linked {
					preview.Links = append(preview.Links, occurrence.Path)
				}
			}
		}
	}
	return preview, nil
}
func (l *Library) Trash(id, path, expected string) (TrashReceipt, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	preview, err := l.planTrash(id, path)
	if err != nil {
		return TrashReceipt{}, err
	}
	if expected == "" || preview.Token != expected {
		return TrashReceipt{}, fmt.Errorf("files changed; review the Trash action again")
	}
	return l.trashItem(preview)
}

func (l *Library) trashItem(preview TrashPreview) (TrashReceipt, error) {
	// Use an unpredictable sibling name so the native path API cannot pick up a replacement at the original name.
	absolute := preview.Path
	parent, err := openAbsoluteDirectory(filepath.Dir(absolute))
	if err != nil {
		return TrashReceipt{}, err
	}
	defer parent.Close()
	parentIdentity, _ := descriptorIdentity(parent)
	name := filepath.Base(absolute)
	stage, err := randomName(".skillview-trash-")
	if err != nil {
		return TrashReceipt{}, err
	}
	if err = unix.RenameatxNp(int(parent.Fd()), name, int(parent.Fd()), stage, unix.RENAME_EXCL); err != nil {
		return TrashReceipt{}, err
	}
	staged := filepath.Join(filepath.Dir(absolute), stage)
	stat, err := statAt(parent, stage)
	if err != nil || fmt.Sprintf("%d:%d", stat.Dev, stat.Ino) != preview.identity {
		if restoreErr := unix.RenameatxNp(int(parent.Fd()), stage, int(parent.Fd()), name, unix.RENAME_EXCL); restoreErr != nil {
			return TrashReceipt{}, fmt.Errorf("item changed; staged item is kept at %s", staged)
		}
		return TrashReceipt{}, fmt.Errorf("files changed; review the Trash action again")
	}
	mover := l.trash
	if mover == nil {
		mover = macTrash
	}
	destination, err := mover(staged)
	if err != nil {
		if restoreErr := unix.RenameatxNp(int(parent.Fd()), stage, int(parent.Fd()), name, unix.RENAME_EXCL); restoreErr != nil {
			return TrashReceipt{}, fmt.Errorf("Trash failed; original content is kept at %s", staged)
		}
		return TrashReceipt{}, fmt.Errorf("could not move this item to Trash; original kept")
	}
	// Restore the visible filename in Trash without replacing a previous item.
	if destination != "" {
		target := filepath.Join(filepath.Dir(destination), name)
		if _, err := os.Lstat(target); !os.IsNotExist(err) {
			suffix, _ := randomName("-")
			target += suffix
		}
		if _, err := os.Lstat(target); os.IsNotExist(err) {
			if unix.RenamexNp(destination, target, unix.RENAME_EXCL) == nil {
				destination = target
			}
		}
	}
	_ = parent.Sync()
	if resolved, resolveErr := filepath.EvalSymlinks(filepath.Dir(destination)); resolveErr == nil {
		destination = filepath.Join(resolved, filepath.Base(destination))
	}
	receiptID, _ := randomName("trash-")
	receipt := TrashReceipt{ID: receiptID, OriginalPath: absolute, TrashPath: destination, ParentIdentity: parentIdentity}
	if folder, err := openAbsoluteDirectory(filepath.Dir(destination)); err == nil {
		if item, err := statAt(folder, filepath.Base(destination)); err == nil {
			receipt.Identity = fmt.Sprintf("%d:%d", item.Dev, item.Ino)
		}
		folder.Close()
	}
	l.prefs.RecentTrash = append(l.prefs.RecentTrash, receipt)
	if len(l.prefs.RecentTrash) > 50 {
		l.prefs.RecentTrash = l.prefs.RecentTrash[len(l.prefs.RecentTrash)-50:]
	}
	if err := l.persist(); err != nil {
		receipt.Warning = "Item moved to Trash; recovery history could not be saved."
	}
	return receipt, nil
}

// Limited native picker input: ordinary files only, never execute their contents.
func ReadImport(path string) ([]byte, error) {
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC|unix.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(fd), path)
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return nil, fmt.Errorf("choose an ordinary file")
	}
	if info.Size() > 16*1024*1024 {
		return nil, fmt.Errorf("choose a resource smaller than 16 MiB")
	}
	return io.ReadAll(io.LimitReader(file, 16*1024*1024+1))
}

func (l *Library) UndoTrash(id string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	index := -1
	var receipt TrashReceipt
	for i, item := range l.prefs.RecentTrash {
		if item.ID == id {
			index = i
			receipt = item
			break
		}
	}
	if index < 0 || receipt.TrashPath == "" || receipt.Identity == "" {
		return fmt.Errorf("this item is no longer available for recovery")
	}
	source, err := openAbsoluteDirectory(filepath.Dir(receipt.TrashPath))
	if err != nil {
		return err
	}
	defer source.Close()
	stat, err := statAt(source, filepath.Base(receipt.TrashPath))
	if err != nil {
		return err
	}
	identity := fmt.Sprintf("%d:%d", stat.Dev, stat.Ino)
	if identity != receipt.Identity {
		return fmt.Errorf("Trash item changed; restore it in Finder")
	}
	target, err := openAbsoluteDirectory(filepath.Dir(receipt.OriginalPath))
	if err != nil {
		return err
	}
	defer target.Close()
	parentID, _ := descriptorIdentity(target)
	if parentID != receipt.ParentIdentity {
		return fmt.Errorf("original folder changed; restore this item in Finder")
	}
	if err = unix.RenameatxNp(int(source.Fd()), filepath.Base(receipt.TrashPath), int(target.Fd()), filepath.Base(receipt.OriginalPath), unix.RENAME_EXCL); err != nil {
		return fmt.Errorf("could not restore without replacing an existing item; use Finder")
	}
	l.prefs.RecentTrash = append(l.prefs.RecentTrash[:index], l.prefs.RecentTrash[index+1:]...)
	_ = source.Sync()
	_ = target.Sync()
	return l.persist()
}

func (l *Library) ResourceReferences(id, target string) ([]string, error) {
	resources, err := l.Resources(id)
	if err != nil {
		return nil, err
	}
	result := []string{}
	for _, resource := range resources {
		if resource.Kind != "markdown" {
			continue
		}
		doc, err := l.Read(id, resource.Path)
		if err != nil {
			continue
		}
		found := false
		tree := goldmark.DefaultParser().Parse(text.NewReader([]byte(doc.Body)))
		_ = ast.Walk(tree, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
			if !entering {
				return ast.WalkContinue, nil
			}
			var destination []byte
			switch link := node.(type) {
			case *ast.Link:
				destination = link.Destination
			case *ast.Image:
				destination = link.Destination
			}
			u, err := url.Parse(string(destination))
			if err == nil && u.Scheme == "" && u.Host == "" && u.Path != "" {
				resolved := filepath.Clean(filepath.Join(filepath.Dir(resource.Path), u.Path))
				if resolved == target {
					found = true
				}
			}
			return ast.WalkContinue, nil
		})
		if found {
			result = append(result, resource.Path)
		}
	}
	return result, nil
}
