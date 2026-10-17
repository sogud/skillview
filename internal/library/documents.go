package library

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/sys/unix"
)

func (l *Library) Resources(id string) ([]Resource, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	directory, _, err := l.openSkill(id)
	if err != nil {
		return nil, err
	}
	defer directory.Close()
	if ref := l.refs[id]; ref.fileName != "" {
		file, err := openChild(directory, ref.fileName, false)
		if err != nil {
			return nil, err
		}
		defer file.Close()
		info, err := file.Stat()
		if err != nil {
			return nil, err
		}
		return []Resource{{Path: ref.fileName, Kind: "markdown", Size: info.Size()}}, nil
	}
	return directoryResources(directory, false)
}

func directoryResources(directory *os.File, skipGenerated bool) ([]Resource, error) {
	resources := []Resource{}
	var walk func(*os.File, string, int) error
	walk = func(parent *os.File, prefix string, depth int) error {
		if depth > 64 || len(resources) > 10000 {
			return fmt.Errorf("resource tree is too large; inspect this folder in Finder")
		}
		entries, err := parent.ReadDir(-1)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if entry.Name() == ".git" || (skipGenerated && entry.IsDir() && skipDiscoveryDirectory(entry.Name())) {
				continue
			}
			path := filepath.Join(prefix, entry.Name())
			if entry.Type()&os.ModeSymlink != 0 {
				resources = append(resources, Resource{Path: path, Kind: "link"})
				continue
			}
			child, err := openChild(parent, entry.Name(), entry.IsDir())
			if err != nil {
				return fmt.Errorf("%s changed; refresh resources: %w", path, err)
			}
			if entry.IsDir() {
				err = walk(child, path, depth+1)
				child.Close()
				if err != nil {
					return err
				}
				continue
			}
			info, err := child.Stat()
			revision := resourceRevision(child)
			child.Close()
			if err != nil {
				return err
			}
			kind := resourceKind(path)
			if !info.Mode().IsRegular() {
				kind = "binary"
			}
			resources = append(resources, Resource{Path: path, Kind: kind, Size: info.Size(), Revision: revision})
		}
		return nil
	}
	if err := walk(directory, "", 0); err != nil {
		return nil, err
	}
	sort.Slice(resources, func(i, j int) bool { return resources[i].Path < resources[j].Path })
	return resources, nil
}

func resourceKind(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md", ".markdown":
		return "markdown"
	case ".png", ".jpg", ".jpeg", ".gif", ".webp":
		return "image"
	default:
		return "text"
	}
}

func openResource(directory *os.File, path string) (*os.File, *os.File, string, error) {
	if err := validateRelative(path); err != nil {
		return nil, nil, "", err
	}
	parentPath := filepath.Dir(path)
	if parentPath == "." {
		parentPath = ""
	}
	parent, err := openDirectory(directory, parentPath)
	if err != nil {
		return nil, nil, "", fmt.Errorf("resource directory changed or contains a link: %w", err)
	}
	name := filepath.Base(path)
	file, err := openChild(parent, name, false)
	if err != nil {
		parent.Close()
		return nil, nil, "", fmt.Errorf("file changed or is a link; refresh the resource list: %w", err)
	}
	return file, parent, name, nil
}

func (l *Library) Read(id, path string) (Document, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.read(id, path)
}

func (l *Library) read(id, path string) (Document, error) {
	directory, source, err := l.openSkill(id)
	if err != nil {
		return Document{}, err
	}
	defer directory.Close()
	flat := l.refs[id].fileName
	if flat != "" {
		if path == "SKILL.md" {
			path = flat
		}
		if path != flat {
			return Document{}, fmt.Errorf("standalone Skills expose only their own document")
		}
		source.Writable = false
	}
	file, parent, _, err := openResource(directory, path)
	if err != nil {
		return Document{}, err
	}
	defer file.Close()
	defer parent.Close()
	doc, err := document(file, id, path, source.Writable && l.skillEditable(id))
	if flat != "" {
		_, body, _ := frontmatter([]byte(doc.Text))
		doc.Body = string(body)
	}
	return doc, err
}

func (l *Library) skillEditable(id string) bool {
	for _, skill := range l.catalog.Skills {
		if skill.ID == id {
			return skill.Editable
		}
	}
	return false
}

func document(file *os.File, id, path string, writable bool) (Document, error) {
	info, err := file.Stat()
	if err != nil {
		return Document{}, err
	}
	doc := Document{SkillID: id, Path: path, Size: info.Size(), Kind: resourceKind(path), Newline: "\n"}
	if !info.Mode().IsRegular() {
		doc.Kind = "binary"
		doc.ReadOnlyReason = "Only regular files can be read or edited."
		return doc, nil
	}
	if doc.Kind == "image" {
		if info.Size() > 8*1024*1024 {
			doc.Kind = "binary"
			doc.ReadOnlyReason = "Image is larger than the 8 MiB preview limit."
			return doc, nil
		}
		data, err := io.ReadAll(io.LimitReader(file, 8*1024*1024+1))
		if err != nil {
			return doc, err
		}
		mime := http.DetectContentType(data)
		if !hasString([]string{"image/png", "image/jpeg", "image/gif", "image/webp"}, mime) {
			doc.Kind = "binary"
			return doc, nil
		}
		doc.DataURL = "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data)
		doc.Revision = digest(data)
		return doc, nil
	}
	data, err := readRegularText(file)
	if err != nil {
		doc.Kind = "binary"
		doc.ReadOnlyReason = err.Error()
		return doc, nil
	}
	doc.Text = string(data)
	doc.Body = doc.Text
	doc.Revision = digest(data)
	if path == "SKILL.md" {
		_, body, _ := frontmatter(data)
		doc.Body = string(body)
	}
	if strings.Contains(doc.Text, "\r\n") {
		doc.Newline = "\r\n"
	}
	links, err := fileLinks(file)
	if err != nil {
		return doc, err
	}
	doc.Editable = writable && links == 1
	if !writable {
		doc.ReadOnlyReason = "This source or linked entry is read-only. Select an original folder with editing allowed."
	}
	if links != 1 {
		doc.ReadOnlyReason = "This file has hard links. Edit it outside SkillView to preserve its shared identity."
	}
	return doc, nil
}

func (l *Library) RevealPath(id, path string) (string, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	directory, source, err := l.openSkill(id)
	if err != nil {
		return "", err
	}
	defer directory.Close()
	if flat := l.refs[id].fileName; flat != "" {
		if path == "" || path == "SKILL.md" {
			path = flat
		}
		if path != flat {
			return "", fmt.Errorf("standalone Skills expose only their own document")
		}
	}
	if path != "" {
		file, parent, _, err := openResource(directory, path)
		if err != nil {
			return "", err
		}
		file.Close()
		parent.Close()
	}
	return filepath.Join(source.ResolvedPath, l.refs[id].relativeDir, path), nil
}

func statAt(parent *os.File, name string) (unix.Stat_t, error) {
	var stat unix.Stat_t
	err := unix.Fstatat(int(parent.Fd()), name, &stat, unix.AT_SYMLINK_NOFOLLOW)
	return stat, err
}
