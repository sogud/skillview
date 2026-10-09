package app

import (
	"context"
	"fmt"
	"net/url"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"skillview/internal/library"
	"skillview/internal/locale"
	"skillview/internal/translation"
	"skillview/internal/updates"
)

type Native struct {
	Application    func() *application.App
	Window         func() *application.WebviewWindow
	DiscoveryRoots []string
	UpdateLanguage func(string)
	Updates        func() *updates.Manager
}

type Service struct {
	library     *library.Library
	native      Native
	mu          sync.Mutex
	ready       bool
	allowQuit   bool
	discoveryMu sync.Mutex
	translator  *translation.Service
}

func New(l *library.Library, native Native, translator *translation.Service) *Service {
	return &Service{library: l, native: native, translator: translator}
}

func (s *Service) Scan(ctx context.Context) (library.Catalog, error) {
	s.discoveryMu.Lock()
	defer s.discoveryMu.Unlock()
	return s.library.Scan(ctx, s.scanReporter())
}
func (s *Service) Discover(ctx context.Context) (library.Catalog, error) {
	s.discoveryMu.Lock()
	defer s.discoveryMu.Unlock()
	if len(s.native.DiscoveryRoots) > 0 {
		if err := s.library.Discover(ctx, s.native.DiscoveryRoots, s.scanReporter()); err != nil {
			return library.Catalog{}, err
		}
	}
	return s.library.Scan(ctx, s.scanReporter())
}

func (s *Service) scanReporter() func(library.ScanProgress) {
	last := time.Time{}
	phase := ""
	return func(progress library.ScanProgress) {
		finished := progress.Phase == "reading" && progress.SourcesTotal > 0 && progress.SourcesDone == progress.SourcesTotal
		if progress.Phase == phase && time.Since(last) < 100*time.Millisecond && !finished {
			return
		}
		last, phase = time.Now(), progress.Phase
		if s.native.Application != nil {
			s.native.Application().Event.Emit("skillview:scan", progress)
		}
	}
}
func (s *Service) SystemLanguage() string { return locale.SystemLanguage() }
func (s *Service) SetLanguage(language string) error {
	if err := s.library.SetLanguage(language); err != nil {
		return err
	}
	if s.native.UpdateLanguage != nil {
		s.native.UpdateLanguage(language)
	}
	return nil
}
func (s *Service) TranslationSettings() (translation.Settings, error) { return s.translator.Settings() }
func (s *Service) TestTranslationConnection(ctx context.Context) error {
	return s.translator.TestConnection(ctx)
}
func (s *Service) SetTranslationSettings(settings translation.Settings, aiKey, googleKey string, clearAI, clearGoogle bool) error {
	return s.translator.Update(settings, aiKey, googleKey, clearAI, clearGoogle)
}
func (s *Service) Translate(ctx context.Context, id, path, revision, requestID string) (translation.Result, error) {
	doc, err := s.library.Read(id, path)
	if err != nil {
		return translation.Result{}, err
	}
	if doc.Revision != revision {
		return translation.Result{}, fmt.Errorf("This file changed. Reopen it before translating.")
	}
	if doc.Kind != "markdown" && doc.Kind != "text" {
		return translation.Result{}, fmt.Errorf("only text documents can be translated")
	}
	result, err := s.translator.Translate(ctx, doc.Body, revision, func(progress translation.Progress) {
		progress.RequestID = requestID
		if s.native.Application != nil {
			s.native.Application().Event.Emit("skillview:translation", progress)
		}
	})
	if err != nil {
		return translation.Result{}, err
	}
	current, err := s.library.Read(id, path)
	if err != nil {
		return translation.Result{}, err
	}
	if current.Revision != revision {
		return translation.Result{}, fmt.Errorf("This file changed. Reopen it before translating.")
	}
	return result, nil
}
func (s *Service) ExportTranslation(id, path string, result translation.Result) (bool, error) {
	doc, err := s.library.Read(id, path)
	if err != nil {
		return false, err
	}
	if doc.Revision != result.Revision {
		return false, fmt.Errorf("This file changed. Reopen it before translating.")
	}
	language := s.library.Settings().Language
	name := filepath.Base(doc.Path)
	extension := filepath.Ext(name)
	filename := strings.TrimSuffix(name, extension) + ".translated" + extension
	destination, err := s.native.Application().Dialog.SaveFile().
		SetFilename(filename).
		SetMessage(locale.T(language, "Export translation to a new file. Original files will be kept.") + " (" + result.TargetLanguage + ")").
		SetButtonText(locale.T(language, "Export translation")).
		CanCreateDirectories(true).
		AttachToWindow(s.native.Window()).PromptForSingleSelection()
	if err != nil || destination == "" {
		return false, err
	}
	err = s.library.Export(id, path, result.Revision, destination, result.Text)
	return err == nil, err
}
func (s *Service) CopyTranslation(id, path string, result translation.Result) error {
	doc, err := s.library.Read(id, path)
	if err != nil {
		return err
	}
	if doc.Revision != result.Revision {
		return fmt.Errorf("This file changed. Reopen it before translating.")
	}
	if len(result.Text) > 2*1024*1024 {
		return fmt.Errorf("translation response is too large")
	}
	text := locale.T(s.library.Settings().Language, "Translation") + " (" + result.TargetLanguage + ")\n\n" + result.Text
	if !s.native.Application().Clipboard.SetText(text) {
		return fmt.Errorf("translation could not be copied")
	}
	return nil
}
func (s *Service) CachedCatalog() (library.Catalog, error)         { return s.library.CachedCatalog() }
func (s *Service) ClearCache() error                               { return s.translator.ClearCache() }
func (s *Service) Read(id, path string) (library.Document, error)  { return s.library.Read(id, path) }
func (s *Service) Resources(id string) ([]library.Resource, error) { return s.library.Resources(id) }
func (s *Service) Save(id, path, revision, text string) (library.SaveResult, error) {
	return s.library.Save(id, path, revision, text)
}
func (s *Service) Create(sourceID, name, description string) (string, error) {
	return s.library.Create(sourceID, name, description)
}
func (s *Service) SetFavorite(id string, value bool) error { return s.library.SetFavorite(id, value) }
func (s *Service) SetFavorites(ids []string, value bool) error {
	return s.library.SetFavorites(ids, value)
}
func (s *Service) Preferences() library.Preferences { return s.library.Settings() }
func (s *Service) SetTheme(theme string) error      { return s.library.SetTheme(theme) }
func (s *Service) RemoveSource(id string) error     { return s.library.RemoveSource(id) }

func (s *Service) ChooseSource(writable bool) (bool, error) {
	language := s.library.Settings().Language
	message := "Select a folder containing Skills. Files will stay in this folder."
	if writable {
		message = "Allow SkillView to edit and create files in the selected original folder. Linked entries stay read-only."
	}
	path, err := s.native.Application().Dialog.OpenFile().CanChooseDirectories(true).CanChooseFiles(false).CanCreateDirectories(false).ShowHiddenFiles(true).SetTitle(locale.T(language, "Add Skills folder")).SetMessage(locale.T(language, message)).SetButtonText(locale.T(language, "Add folder")).AttachToWindow(s.native.Window()).PromptForSingleSelection()
	if err != nil {
		return false, err
	}
	if path == "" {
		return false, nil
	}
	_, err = s.library.AddSource(path, writable)
	return err == nil, err
}

func (s *Service) Reveal(id, path string) error {
	resolved, err := s.library.RevealPath(id, path)
	if err != nil {
		return err
	}
	return exec.Command("/usr/bin/open", "-R", resolved).Run()
}

func (s *Service) OpenLink(raw string) error {
	link, err := url.Parse(raw)
	if err != nil || (link.Scheme != "https" && link.Scheme != "http") || link.Host == "" || link.User != nil {
		return fmt.Errorf("only ordinary HTTP or HTTPS links can be opened")
	}
	return s.native.Application().Browser.OpenURL(link.String())
}

func (s *Service) Ready() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ready {
		return
	}
	s.ready = true
	if s.native.Updates != nil {
		s.native.Updates().Start()
	}
}
func (s *Service) Close() {
	if s.native.Updates != nil {
		s.native.Updates().Close()
	}
}

func (s *Service) ShouldQuit() bool {
	s.mu.Lock()
	allow := s.allowQuit || !s.ready
	s.mu.Unlock()
	if !allow {
		s.native.Application().Event.Emit("skillview:command", "quit")
	}
	return allow
}
func (s *Service) Quit() {
	s.mu.Lock()
	s.allowQuit = true
	s.mu.Unlock()
	s.native.Application().Quit()
}

func (s *Service) UpdateStatus() updates.Status { return s.native.Updates().Status() }

func (s *Service) CheckUpdates(ctx context.Context) updates.Status {
	return s.native.Updates().Check(ctx)
}

func (s *Service) DownloadUpdate(ctx context.Context) updates.Status {
	return s.native.Updates().Download(ctx)
}

// The frontend calls this only after the existing save/discard dialog resolves.
func (s *Service) RestartUpdate(ctx context.Context) error {
	s.mu.Lock()
	s.allowQuit = true
	s.mu.Unlock()
	if err := s.native.Updates().Restart(ctx); err != nil {
		s.mu.Lock()
		s.allowQuit = false
		s.mu.Unlock()
		return err
	}
	return nil
}

func (s *Service) CreateResource(id, path, text string) (library.Document, error) {
	return s.library.CreateResource(id, path, text)
}
func (s *Service) RenameResource(id, path, newPath, revision string) error {
	return s.library.RenameResource(id, path, newPath, revision)
}
func (s *Service) PlanTrash(id, path string) (library.TrashPreview, error) {
	return s.library.PlanTrash(id, path)
}
func (s *Service) PlanLinkTrash(id, sourceID, path string) (library.TrashPreview, error) {
	return s.library.PlanLinkTrash(id, sourceID, path)
}
func (s *Service) TrashLink(id, sourceID, path, token string) (library.TrashReceipt, error) {
	return s.library.TrashLink(id, sourceID, path, token)
}
func (s *Service) Trash(id, path, token string) (library.TrashReceipt, error) {
	return s.library.Trash(id, path, token)
}
func (s *Service) ChooseResource(id string) (string, error) {
	language := s.library.Settings().Language
	path, err := s.native.Application().Dialog.OpenFile().CanChooseFiles(true).CanChooseDirectories(false).SetTitle(locale.T(language, "Import resource")).AttachToWindow(s.native.Window()).PromptForSingleSelection()
	if err != nil || path == "" {
		return "", err
	}
	data, err := library.ReadImport(path)
	if err != nil {
		return "", err
	}
	name := filepath.Base(path)
	return name, s.library.ImportResource(id, name, data)
}

func (s *Service) UndoTrash(id string) error { return s.library.UndoTrash(id) }

func (s *Service) ResourceReferences(id, path string) ([]string, error) {
	return s.library.ResourceReferences(id, path)
}
