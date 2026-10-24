package library

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

// Inspect only the final link entry inside a selected writable source. Parent
// links need their own original directory selection; target content is unused.
func inspectLink(source Source, path string) (TrashPreview, error) {
	if !source.Approved || !source.Writable {
		return TrashPreview{}, fmt.Errorf("select the link's containing folder with editing allowed")
	}
	relative, inside := within(source.Path, path)
	if !inside || relative == "" {
		return TrashPreview{}, fmt.Errorf("select the link's containing folder with editing allowed")
	}
	if err := validateRelative(relative); err != nil {
		return TrashPreview{}, err
	}
	root, err := openSource(source)
	if err != nil {
		return TrashPreview{}, err
	}
	defer root.Close()
	parentPath := filepath.Dir(relative)
	if parentPath == "." {
		parentPath = ""
	}
	parent, err := openDirectory(root, parentPath)
	if err != nil {
		return TrashPreview{}, err
	}
	defer parent.Close()
	name := filepath.Base(relative)
	return inspectLinkAt(parent, name, filepath.Join(source.ResolvedPath, relative))
}

func inspectLinkAt(parent *os.File, name, path string) (TrashPreview, error) {
	stat, err := statAt(parent, name)
	if err != nil {
		return TrashPreview{}, err
	}
	if stat.Mode&unix.S_IFMT != unix.S_IFLNK {
		return TrashPreview{}, fmt.Errorf("this entry is no longer a symbolic link; rescan before removing it")
	}
	target := make([]byte, 4096)
	count, err := unix.Readlinkat(int(parent.Fd()), name, target)
	if err != nil || count == len(target) {
		return TrashPreview{}, fmt.Errorf("link target could not be read")
	}
	value := string(target[:count])
	identity := fmt.Sprintf("%d:%d", stat.Dev, stat.Ino)
	token := digest([]byte(fmt.Sprintf("%s:%d:%d:%d:%d:%d:%s", identity, stat.Size, stat.Mtim.Sec, stat.Mtim.Nsec, stat.Ctim.Sec, stat.Ctim.Nsec, value)))
	return TrashPreview{Path: path, Count: 1, Token: token, Link: true, LinkTarget: value, identity: identity}, nil
}

func (l *Library) PlanLinkTrash(id, sourceID, path string) (TrashPreview, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := l.reloadPreferences(); err != nil {
		return TrashPreview{}, err
	}
	return l.planLinkTrash(id, sourceID, path)
}

func (l *Library) planLinkTrash(id, sourceID, path string) (TrashPreview, error) {
	var occurrence *Occurrence
	for _, skill := range l.catalog.Skills {
		if skill.ID != id {
			continue
		}
		for _, item := range skill.Occurrences {
			if item.SourceID == sourceID && item.Path == path && item.Linked && item.Removable {
				copy := item
				occurrence = &copy
				break
			}
		}
		break
	}
	if occurrence == nil {
		return TrashPreview{}, fmt.Errorf("select the link's containing folder with editing allowed")
	}
	source, err := l.source(sourceID)
	if err != nil {
		return TrashPreview{}, err
	}
	preview, err := inspectLink(source, path)
	if err != nil {
		return TrashPreview{}, err
	}
	if preview.Token != occurrence.LinkRevision {
		return TrashPreview{}, fmt.Errorf("link changed; rescan before removing it")
	}
	return preview, nil
}

func (l *Library) TrashLink(id, sourceID, path, expected string) (TrashReceipt, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	release, err := l.beginWrite()
	if err != nil {
		return TrashReceipt{}, err
	}
	defer release()
	preview, err := l.planLinkTrash(id, sourceID, path)
	if err != nil {
		return TrashReceipt{}, err
	}
	if expected == "" || expected != preview.Token {
		return TrashReceipt{}, fmt.Errorf("link changed; rescan before removing it")
	}
	return l.trashItem(preview)
}
