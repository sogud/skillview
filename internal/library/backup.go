package library

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sys/unix"
)

// Backup captures catalogued originals once. Resource links remain links.
func (l *Library) Backup() (string, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	release, err := l.beginWrite()
	if err != nil {
		return "", err
	}
	defer release()
	directory := filepath.Join(l.dataDir, "backups")
	if err = os.MkdirAll(directory, 0700); err != nil {
		return "", err
	}
	parent, err := openAbsoluteDirectory(directory)
	if err != nil {
		return "", err
	}
	defer parent.Close()
	name, err := randomName("skills-" + time.Now().UTC().Format("20060102T150405Z") + "-")
	if err != nil {
		return "", err
	}
	name += ".tar.gz"
	stage := name + ".partial"
	fd, err := unix.Openat(int(parent.Fd()), stage, unix.O_CREAT|unix.O_EXCL|unix.O_WRONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0600)
	if err != nil {
		return "", err
	}
	file := os.NewFile(uintptr(fd), stage)
	defer file.Close()
	compressed := gzip.NewWriter(file)
	defer compressed.Close()
	writer := tar.NewWriter(compressed)
	defer writer.Close()
	count := 0
	var walk func(*os.File, string, int) error
	walk = func(root *os.File, prefix string, depth int) error {
		if depth > 64 || count > 100000 {
			return fmt.Errorf("Backup is too large.")
		}
		entries, err := root.ReadDir(-1)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			count++
			stat, err := statAt(root, entry.Name())
			if err != nil {
				return err
			}
			path := filepath.Join(prefix, entry.Name())
			kind := stat.Mode & unix.S_IFMT
			header := &tar.Header{Name: filepath.ToSlash(path), Mode: int64(stat.Mode & 0777), ModTime: time.Unix(stat.Mtim.Sec, stat.Mtim.Nsec)}
			if kind == unix.S_IFLNK {
				buffer := make([]byte, 4096)
				n, err := unix.Readlinkat(int(root.Fd()), entry.Name(), buffer)
				if err != nil || n == len(buffer) {
					return fmt.Errorf("Backup link could not be read.")
				}
				header.Typeflag, header.Linkname = tar.TypeSymlink, string(buffer[:n])
				if err = writer.WriteHeader(header); err != nil {
					return err
				}
				continue
			}
			if kind != unix.S_IFREG && kind != unix.S_IFDIR {
				return fmt.Errorf("Backup contains a special file: %s", path)
			}
			child, err := openChild(root, entry.Name(), kind == unix.S_IFDIR)
			if err != nil {
				return err
			}
			if kind == unix.S_IFDIR {
				header.Typeflag = tar.TypeDir
				err = writer.WriteHeader(header)
				if err == nil {
					err = walk(child, path, depth+1)
				}
			} else {
				header.Typeflag, header.Size = tar.TypeReg, stat.Size
				err = writer.WriteHeader(header)
				if err == nil {
					_, err = io.CopyN(writer, child, stat.Size)
				}
				var current unix.Stat_t
				if err == nil {
					err = unix.Fstat(int(child.Fd()), &current)
				}
				if err == nil && (stat.Ino != current.Ino || stat.Size != current.Size || stat.Mtim != current.Mtim || stat.Ctim != current.Ctim) {
					err = fmt.Errorf("Files changed during backup. The partial backup was kept.")
				}
			}
			child.Close()
			if err != nil {
				return err
			}
		}
		return nil
	}
	manifest := []Skill{}
	seen := map[string]bool{}
	for _, skill := range l.catalog.Skills {
		if seen[skill.Path] {
			continue
		}
		seen[skill.Path] = true
		root, _, err := l.openSkill(skill.ID)
		if err != nil {
			return "", err
		}
		if flat := l.refs[skill.ID].fileName; flat != "" {
			var file *os.File
			file, err = openChild(root, flat, false)
			if err == nil {
				var info os.FileInfo
				info, err = file.Stat()
				if err == nil && !info.Mode().IsRegular() {
					err = fmt.Errorf("Backup requires an ordinary file.")
				}
				if err == nil {
					err = writer.WriteHeader(&tar.Header{Name: filepath.ToSlash(filepath.Join("skills", skill.ID, flat)), Mode: int64(info.Mode().Perm()), Size: info.Size()})
				}
				if err == nil {
					_, err = io.CopyN(writer, file, info.Size())
				}
				file.Close()
			}
		} else {
			err = walk(root, filepath.Join("skills", skill.ID), 0)
		}
		root.Close()
		if err != nil {
			return "", err
		}
		manifest = append(manifest, skill)
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return "", err
	}
	if err = writer.WriteHeader(&tar.Header{Name: "manifest.json", Mode: 0600, Size: int64(len(data))}); err != nil {
		return "", err
	}
	if _, err = writer.Write(data); err != nil {
		return "", err
	}
	if err = writer.Close(); err != nil {
		return "", err
	}
	if err = compressed.Close(); err != nil {
		return "", err
	}
	if err = file.Sync(); err != nil {
		return "", err
	}
	if err = unix.RenameatxNp(int(parent.Fd()), stage, int(parent.Fd()), name, unix.RENAME_EXCL); err != nil {
		return "", err
	}
	return filepath.Join(directory, name), parent.Sync()
}
