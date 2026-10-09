package library

// Cached references retain the scanned directory identity. Opening a cached
// item still checks current source permission, links and file identity on disk.
type cachedReference struct {
	SourceID    string  `json:"sourceID"`
	RelativeDir string  `json:"relativeDir"`
	Identity    string  `json:"identity"`
	LinkedRoot  *Source `json:"linkedRoot,omitempty"`
	FileName    string  `json:"fileName,omitempty"`
}

type catalogSnapshot struct {
	Catalog
	References map[string]cachedReference `json:"references"`
}

func snapshotCatalog(catalog Catalog, refs map[string]fileRef) catalogSnapshot {
	snapshot := catalogSnapshot{Catalog: catalog, References: map[string]cachedReference{}}
	for id, ref := range refs {
		snapshot.References[id] = cachedReference{SourceID: ref.sourceID, RelativeDir: ref.relativeDir, Identity: ref.identity, LinkedRoot: ref.linkedRoot, FileName: ref.fileName}
	}
	return snapshot
}
