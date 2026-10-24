package library

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/sys/unix"
)

const maxTextBytes = 2 * 1024 * 1024

type scanState struct {
	library          *Library
	catalog          Catalog
	refs             map[string]fileRef
	positions        map[string]int
	directories      int
	context          context.Context
	progress         []func(ScanProgress)
	sourcesDone      int
	archiveDirectory string
}

func (l *Library) Scan(ctx context.Context, progress ...func(ScanProgress)) (Catalog, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	release, err := l.beginWrite()
	if err != nil {
		return Catalog{}, err
	}
	defer release()
	l.watchPaths = map[string]bool{}
	state := scanState{library: l, catalog: Catalog{Skills: []Skill{}, Sources: []Source{}, Issues: []Issue{}, ScannedAt: time.Now().UTC().Format(time.RFC3339)}, refs: map[string]fileRef{}, positions: map[string]int{}, context: ctx}
	state.archiveDirectory, _ = l.archiveDirectory()
	state.catalog.Discovery = l.discovery
	state.progress = progress
	state.report()
	if home, err := os.UserHomeDir(); err == nil {
		state.catalog.Agents = Agents(home)
	}
	for _, source := range l.prefs.Sources {
		if err := ctx.Err(); err != nil {
			return Catalog{}, err
		}
		l.watchPaths[filepath.Dir(source.ResolvedPath)] = true
		root, err := openSource(source)
		if err != nil {
			source.Status = "unavailable"
			source.Message = err.Error()
			if !source.Approved {
				source.Status = "needs-access"
				source.Message = "Select this directory to allow reading its original files."
				if resolved, e := filepath.EvalSymlinks(source.Path); e == nil {
					source.ResolvedPath = resolved
				} else if errors.Is(e, os.ErrNotExist) {
					source.Status = "missing"
					source.Message = "Directory not found. It has not been created."
				}
			}
			state.catalog.Sources = append(state.catalog.Sources, source)
			state.sourcesDone++
			state.report()
			continue
		}
		source.Status = "ready"
		source.Message = ""
		state.catalog.Sources = append(state.catalog.Sources, source)
		rootLinked := sourceContainsUserLink(source.Path)
		state.walk(root, source, source, "", source.Path, rootLinked, map[string]bool{}, 0)
		root.Close()
		state.sourcesDone++
		state.report()
	}
	if err := ctx.Err(); err != nil {
		return Catalog{}, err
	}
	sort.SliceStable(state.catalog.Skills, func(i, j int) bool { return state.catalog.Skills[i].Name < state.catalog.Skills[j].Name })
	if l.cache != nil {
		data, err := json.Marshal(snapshotCatalog(state.catalog, state.refs))
		if err == nil {
			err = l.cache.Put("catalog", "current", data)
		}
		if err != nil {
			state.catalog.Issues = append(state.catalog.Issues, Issue{Message: "Local cache is unavailable; original files remain readable."})
		}
	}
	l.catalog = state.catalog
	l.refs = state.refs
	return state.catalog, nil
}

func (s *scanState) report() {
	reportProgress(s.progress, ScanProgress{Phase: "reading", Directories: s.directories,
		Skills: len(s.catalog.Skills), SourcesDone: s.sourcesDone, SourcesTotal: len(s.library.prefs.Sources)})
}

func (s *scanState) issue(sourceID, path, target, message string) {
	s.catalog.Issues = append(s.catalog.Issues, Issue{SourceID: sourceID, Path: path, Target: target, Message: message})
}

func (s *scanState) walk(directory *os.File, source Source, entrySource Source, relative, entryPath string, linked bool, ancestors map[string]bool, depth int) {
	if s.archiveDirectory != "" {
		if _, inside := within(s.archiveDirectory, filepath.Join(source.ResolvedPath, relative)); inside {
			return
		}
	}
	if s.context.Err() != nil {
		return
	}
	s.directories++
	s.report()
	if depth > 64 || s.directories > 20000 {
		s.issue(source.ID, entryPath, "", "Directory scan limit reached; choose a smaller Skills folder.")
		return
	}
	s.library.watchPaths[filepath.Join(source.ResolvedPath, relative)] = true
	identity, err := descriptorIdentity(directory)
	if err != nil {
		s.issue(source.ID, entryPath, "", err.Error())
		return
	}
	if ancestors[identity] {
		s.issue(source.ID, entryPath, "", "Directory link cycle; scanning stopped here.")
		return
	}
	ancestors[identity] = true
	defer delete(ancestors, identity)
	skillFile, err := openChild(directory, "SKILL.md", false)
	if err == nil {
		data, readErr := readRegularText(skillFile)
		skillFile.Close()
		if readErr != nil {
			s.issue(source.ID, filepath.Join(entryPath, "SKILL.md"), "", readErr.Error())
			return
		}
		canonical := filepath.Join(source.ResolvedPath, relative)
		name, description, warnings := parseSkill(data, canonical)
		occurrence := Occurrence{SourceID: entrySource.ID, Path: entryPath, Linked: linked}
		if linked && entrySource.Writable {
			if preview, err := inspectLink(entrySource, entryPath); err == nil {
				occurrence.Removable = true
				occurrence.LinkRevision = preview.Token
			}
		}
		index, exists := s.positions[identity]
		if exists {
			skill := &s.catalog.Skills[index]
			skill.Occurrences = append(skill.Occurrences, occurrence)
			for _, tool := range entrySource.Tools {
				if !hasString(skill.Tools, tool) {
					skill.Tools = append(skill.Tools, tool)
				}
			}
			if source.Writable && !linked {
				skill.Editable = true
				s.refs[skill.ID] = fileRef{sourceID: source.ID, relativeDir: relative, identity: identity}
			}
		} else {
			id := objectID(canonical)
			skill := Skill{ID: id, ContentKey: skillContentKey(name, data), Name: name, Description: description, Path: canonical, Revision: digest(data), Warnings: warnings, Occurrences: []Occurrence{occurrence}, Tools: append([]string{}, entrySource.Tools...), Editable: source.Writable && !linked}
			s.positions[identity] = len(s.catalog.Skills)
			s.catalog.Skills = append(s.catalog.Skills, skill)
			ref := fileRef{sourceID: source.ID, relativeDir: relative, identity: identity}
			if linked {
				root := source
				root.Writable = false
				ref.linkedRoot = &root
			}
			s.refs[id] = ref
		}
		s.watchResourceDirectories(directory, canonical, 0)
		return // Resource trees are read separately, on demand.
	}
	if !errors.Is(err, os.ErrNotExist) {
		s.issue(source.ID, filepath.Join(entryPath, "SKILL.md"), "", "SKILL.md is not a readable ordinary file: "+err.Error())
	}
	entries, err := directory.ReadDir(-1)
	if err != nil {
		s.issue(source.ID, entryPath, "", err.Error())
		return
	}
	if hasString(entrySource.Tools, "OpenCode") {
		s.scanFlat(directory, source, entrySource, relative, entryPath, linked, identity, entries)
	}
	for _, entry := range entries {
		if s.directories > 20000 || s.context.Err() != nil {
			return
		}
		name := entry.Name()
		if name == ".git" || name == "node_modules" || name == ".artifacts" || name == ".skillview-archive" || strings.HasPrefix(name, ".skillview-archiving-") {
			continue
		}
		childPath := filepath.Join(entryPath, name)
		childRelative := filepath.Join(relative, name)
		if entry.Type()&os.ModeSymlink != 0 {
			target, err := filepath.EvalSymlinks(childPath)
			if err != nil {
				s.issue(source.ID, childPath, "", "Broken or cyclic link: "+err.Error())
				continue
			}
			authorized := false
			for _, candidate := range s.library.prefs.Sources {
				candidateRelative, inside := within(candidate.ResolvedPath, target)
				if !candidate.Approved || !inside {
					continue
				}
				candidateRoot, e := openSource(candidate)
				if e != nil {
					continue
				}
				child, e := openDirectory(candidateRoot, candidateRelative)
				candidateRoot.Close()
				if e != nil {
					continue
				}
				s.walk(child, candidate, entrySource, candidateRelative, childPath, true, ancestors, depth+1)
				child.Close()
				authorized = true
				break
			}
			if !authorized {
				if entrySource.Automatic {
					rootIdentity, e := pathIdentity(target)
					if e == nil {
						linkedSource := Source{ID: entrySource.ID, Path: childPath, ResolvedPath: target, Identity: rootIdentity, Approved: true, Automatic: true, Tools: entrySource.Tools}
						child, e := openSource(linkedSource)
						if e == nil {
							s.walk(child, linkedSource, entrySource, "", childPath, true, ancestors, depth+1)
							child.Close()
							authorized = true
						}
					}
				}
				if !authorized {
					s.issue(source.ID, childPath, target, "Linked target is outside readable sources. Add the original directory to read it.")
				}
			}
			continue
		}
		if !entry.IsDir() {
			continue
		}
		child, err := openChild(directory, name, true)
		if err != nil {
			s.issue(source.ID, childPath, "", err.Error())
			continue
		}
		s.walk(child, source, entrySource, childRelative, childPath, linked, ancestors, depth+1)
		child.Close()
	}
}

func hasString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func readRegularText(file *os.File) ([]byte, error) {
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("not a regular file")
	}
	if info.Size() > maxTextBytes {
		return nil, fmt.Errorf("text file exceeds the 2 MiB reader limit")
	}
	data, err := io.ReadAll(io.LimitReader(file, maxTextBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxTextBytes || !utf8.Valid(data) {
		return nil, fmt.Errorf("file is too large or is not UTF-8 text")
	}
	for _, value := range data {
		if value == 0 {
			return nil, fmt.Errorf("binary content cannot be read as text")
		}
	}
	return data, nil
}

func (l *Library) openSkill(id string) (*os.File, Source, error) {
	ref, ok := l.refs[id]
	if !ok {
		return nil, Source{}, fmt.Errorf("Skill no longer exists; rescan the library")
	}
	source, err := l.source(ref.sourceID)
	if err != nil {
		return nil, Source{}, err
	}
	if ref.linkedRoot != nil {
		linked := *ref.linkedRoot
		if linked.Path != source.Path || linked.ResolvedPath != source.ResolvedPath || linked.Identity != source.Identity {
			_, inside := within(source.Path, linked.Path)
			resolved, resolveErr := filepath.EvalSymlinks(linked.Path)
			if !source.Automatic || linked.ID != source.ID || !inside || resolveErr != nil || resolved != linked.ResolvedPath {
				return nil, Source{}, fmt.Errorf("Skill link changed; rescan the library")
			}
		}
		source = *ref.linkedRoot
		source.Writable = false
	}
	root, err := openSource(source)
	if err != nil {
		return nil, Source{}, err
	}
	directory, err := openDirectory(root, ref.relativeDir)
	root.Close()
	if err != nil {
		return nil, Source{}, fmt.Errorf("Skill directory changed; rescan: %w", err)
	}
	identity, err := descriptorIdentity(directory)
	if err != nil || identity != ref.identity {
		directory.Close()
		return nil, Source{}, fmt.Errorf("Skill directory was replaced; rescan the library")
	}
	return directory, source, nil
}

func fileLinks(file *os.File) (uint16, error) {
	var stat unix.Stat_t
	if err := unix.Fstat(int(file.Fd()), &stat); err != nil {
		return 0, err
	}
	return stat.Nlink, nil
}

func (l *Library) CachedCatalog() (Catalog, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := l.reloadPreferences(); err != nil {
		return Catalog{}, err
	}
	empty := Catalog{Skills: []Skill{}, Sources: l.prefs.Sources, NeedsScan: true}
	if home, err := os.UserHomeDir(); err == nil {
		empty.Agents = Agents(home)
	}
	if l.cache == nil {
		return empty, nil
	}
	data, err := l.cache.Get("catalog", "current")
	if err != nil || len(data) == 0 {
		return empty, err
	}
	var snapshot catalogSnapshot
	if err = json.Unmarshal(data, &snapshot); err != nil {
		return empty, err
	}
	catalog := snapshot.Catalog
	catalog.Agents = empty.Agents
	// A removed source must not reappear through cached metadata.
	for _, source := range catalog.Sources {
		if _, err = l.source(source.ID); err != nil {
			return empty, nil
		}
	}
	refs := map[string]fileRef{}
	for _, skill := range catalog.Skills {
		cached, exists := snapshot.References[skill.ID]
		if !exists {
			catalog.NeedsScan = true
			continue
		}
		source, sourceErr := l.source(cached.SourceID)
		if sourceErr != nil {
			return empty, nil
		}
		if cached.LinkedRoot != nil {
			source = *cached.LinkedRoot
		}
		path := filepath.Join(source.ResolvedPath, cached.RelativeDir, cached.FileName)
		if skill.Path != path || skill.ID != objectID(path) || cached.Identity == "" {
			return empty, nil
		}
		refs[skill.ID] = fileRef{sourceID: cached.SourceID, relativeDir: cached.RelativeDir, identity: cached.Identity, linkedRoot: cached.LinkedRoot, fileName: cached.FileName}
	}
	l.catalog, l.refs, l.discovery = catalog, refs, catalog.Discovery
	return catalog, nil
}

func (s *scanState) watchResourceDirectories(directory *os.File, path string, depth int) {
	if depth > 64 || len(s.library.watchPaths) >= 8000 {
		return
	}
	s.library.watchPaths[path] = true
	entries, err := directory.ReadDir(-1)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if !entry.IsDir() || skipDiscoveryDirectory(entry.Name()) {
			continue
		}
		child, err := openChild(directory, entry.Name(), true)
		if err != nil {
			continue
		}
		s.watchResourceDirectories(child, filepath.Join(path, entry.Name()), depth+1)
		child.Close()
	}
}

// OpenCode also accepts Markdown files directly inside a configured skills root.
// These standalone entries stay read-only and expose only their own document.
func (s *scanState) scanFlat(directory *os.File, source, entrySource Source, relative, entryPath string, linked bool, identity string, entries []os.DirEntry) {
	if relative != "" {
		return
	}
	for _, entry := range entries {
		if !entry.Type().IsRegular() || !strings.EqualFold(filepath.Ext(entry.Name()), ".md") {
			continue
		}
		file, err := openChild(directory, entry.Name(), false)
		if err != nil {
			continue
		}
		data, err := readRegularText(file)
		fileID, _ := descriptorIdentity(file)
		file.Close()
		if err != nil {
			continue
		}
		canonical := filepath.Join(source.ResolvedPath, relative, entry.Name())
		name, description, warnings := parseSkill(data, canonical)
		occurrence := Occurrence{SourceID: entrySource.ID, Path: filepath.Join(entryPath, entry.Name()), Linked: linked}
		if index, exists := s.positions["flat:"+fileID]; exists {
			s.catalog.Skills[index].Occurrences = append(s.catalog.Skills[index].Occurrences, occurrence)
			continue
		}
		id := objectID(canonical)
		s.positions["flat:"+fileID] = len(s.catalog.Skills)
		s.catalog.Skills = append(s.catalog.Skills, Skill{ID: id, ContentKey: skillContentKey(name, data), Name: name, Description: description, Path: canonical, Revision: digest(data), Warnings: warnings, Occurrences: []Occurrence{occurrence}, Tools: append([]string{}, entrySource.Tools...)})
		ref := fileRef{sourceID: source.ID, relativeDir: relative, identity: identity, fileName: entry.Name()}
		if linked {
			source.Writable = false
			ref.linkedRoot = &source
		}
		s.refs[id] = ref
	}
}

// Display grouping ignores line endings and outer whitespace. File revisions
// remain byte-exact, and each physical copy keeps its own access and resources.
func skillContentKey(name string, data []byte) string {
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	return digest([]byte(name + "\x00" + strings.TrimSpace(text)))
}

func sourceContainsUserLink(path string) bool {
	current := "/"
	for _, part := range strings.Split(strings.TrimPrefix(filepath.Clean(path), "/"), "/") {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			return false
		}
		if info.Mode()&os.ModeSymlink != 0 && current != "/var" && current != "/tmp" && current != "/etc" {
			return true
		}
	}
	return false
}
