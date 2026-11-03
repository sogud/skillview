package translation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	kit "github.com/sogud/translate-kit"
	"net/url"
	"os"
	"path/filepath"
	"skillview/internal/cache"
	"strings"
	"sync"
)

type Settings struct {
	DisplayMode    string `json:"displayMode"`
	Provider       string `json:"provider"`
	TargetLanguage string `json:"targetLanguage"`
	BaseURL        string `json:"baseUrl"`
	Model          string `json:"model"`
	HasAIKey       bool   `json:"hasAIKey"`
	HasGoogleKey   bool   `json:"hasGoogleKey"`
	HasLibreKey    bool   `json:"hasLibreKey"`
	LibreURL       string `json:"libreUrl"`
}

type KeyStore interface {
	Get(string) (string, error)
	Set(string, string) error
	Delete(string) error
}

// Additional provider keys are written to Keychain, never to settings JSON.
type KeyChange struct {
	Name  string
	Value string
	Clear bool
}

type Service struct {
	mu       sync.Mutex
	dataDir  string
	settings Settings
	keys     KeyStore
	cache    map[string]Result
	store    *cache.Store
	client   *kit.Client
}
type Result struct {
	Text           string `json:"text"`
	Provider       string `json:"provider"`
	TargetLanguage string `json:"targetLanguage"`
	Revision       string `json:"revision"`
	Cached         bool   `json:"cached"`
	Sections       int    `json:"sections"`
}

func New(dataDir string, keys KeyStore) (*Service, error) {
	s := &Service{dataDir: dataDir, keys: keys, settings: Settings{DisplayMode: "immersive", Provider: "google", TargetLanguage: "zh-CN", BaseURL: "https://api.openai.com/v1"}, client: kit.New(kit.Options{}), cache: map[string]Result{}}
	data, err := os.ReadFile(filepath.Join(dataDir, "translation.json"))
	if err == nil {
		if err = json.Unmarshal(data, &s.settings); err != nil {
			return nil, fmt.Errorf("translation settings could not be read")
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	// Old settings omit this presentation preference; unknown persisted values
	// also fall back to the original immersive reader.
	if s.settings.DisplayMode != "translation" {
		s.settings.DisplayMode = "immersive"
	}
	// Keep old API keys, but retire the paid provider from the active choices.
	if s.settings.Provider == "google-cloud" {
		s.settings.Provider = "google"
	}
	return s, nil
}

func (s *Service) Settings() (Settings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	settings := s.settings
	ai, err := s.keys.Get("ai")
	if err != nil {
		return Settings{}, err
	}
	google, err := s.keys.Get("google")
	if err != nil {
		return Settings{}, err
	}
	settings.HasAIKey = ai != ""
	settings.HasGoogleKey = google != ""
	libre, err := s.keys.Get("libretranslate")
	if err != nil {
		return Settings{}, err
	}
	settings.HasLibreKey = libre != ""
	return settings, nil
}

func (s *Service) Update(settings Settings, aiKey, googleKey string, clearAI, clearGoogle bool, additional ...KeyChange) (err error) {
	if settings.DisplayMode == "" {
		settings.DisplayMode = "immersive"
	}
	if settings.DisplayMode != "immersive" && settings.DisplayMode != "translation" {
		return fmt.Errorf("unknown translation display mode")
	}
	if settings.Provider != "google" && settings.Provider != "google-cloud" && settings.Provider != "ai" && settings.Provider != "libretranslate" && settings.Provider != "youdao" {
		return fmt.Errorf("unknown translation provider")
	}
	if !hasLanguage(settings.TargetLanguage) {
		return fmt.Errorf("unsupported target language")
	}
	if settings.Provider == "ai" {
		if _, err := validateBaseURL(settings.BaseURL); err != nil {
			return err
		}
		if strings.TrimSpace(settings.Model) == "" {
			return fmt.Errorf("choose an AI model in Translation settings")
		}
	}
	if settings.Provider == "libretranslate" {
		if _, err := validateBaseURL(settings.LibreURL); err != nil {
			return err
		}
	}
	changes := []KeyChange{{Name: "ai", Value: aiKey, Clear: clearAI}, {Name: "google", Value: googleKey, Clear: clearGoogle}}
	for _, change := range additional {
		if change.Name != "libretranslate" {
			return fmt.Errorf("unknown translation provider")
		}
		changes = append(changes, change)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	previousKeys := map[string]string{}
	for _, change := range changes {
		if change.Value != "" || change.Clear {
			old, readErr := s.keys.Get(change.Name)
			if readErr != nil {
				return readErr
			}
			previousKeys[change.Name] = old
		}
	}
	defer func() {
		if err == nil {
			return
		}
		for name, old := range previousKeys {
			var restoreErr error
			if old == "" {
				restoreErr = s.keys.Delete(name)
			} else {
				restoreErr = s.keys.Set(name, old)
			}
			if restoreErr != nil {
				err = fmt.Errorf("translation settings failed and the key could not be restored; check Keychain")
			}
		}
	}()
	for _, change := range changes {
		if change.Value != "" {
			if err := s.keys.Set(change.Name, change.Value); err != nil {
				return err
			}
		} else if change.Clear {
			if err := s.keys.Delete(change.Name); err != nil {
				return err
			}
		}
	}
	settings.HasAIKey = false
	settings.HasGoogleKey = false
	settings.HasLibreKey = false
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(s.dataDir, ".translation-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	defer file.Close()
	if _, err = file.Write(data); err != nil {
		return err
	}
	if err = file.Sync(); err != nil {
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	if err = os.Rename(file.Name(), filepath.Join(s.dataDir, "translation.json")); err != nil {
		return err
	}
	previous := s.settings
	previous.DisplayMode = settings.DisplayMode
	// Display changes reuse both in-memory and disk translations. Keys and
	// provider settings still invalidate the in-memory results as before.
	if previous != settings || len(previousKeys) != 0 {
		s.cache = map[string]Result{}
	}
	s.settings = settings
	return nil
}

func hasLanguage(language string) bool             { return kit.SupportsLanguage(language) }
func validateBaseURL(raw string) (*url.URL, error) { return kit.ValidateBaseURL(raw) }
func config(settings Settings) kit.Config {
	return kit.Config{Provider: settings.Provider, TargetLanguage: settings.TargetLanguage, BaseURL: settings.BaseURL, LibreURL: settings.LibreURL, Model: settings.Model}
}
func (s *Service) Translate(ctx context.Context, body, revision string, progress ...func(Progress)) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	if len(body) > 1024*1024 {
		return Result{}, fmt.Errorf("translate a document smaller than 1 MiB")
	}
	s.mu.Lock()
	settings := s.settings
	key := ""
	var err error
	if keyName := providerKeyName(settings.Provider); keyName != "" {
		key, err = s.keys.Get(keyName)
	}
	if err != nil {
		s.mu.Unlock()
		return Result{}, err
	}
	hash := sha256.Sum256([]byte(body + revision + settings.Provider + settings.TargetLanguage + settings.BaseURL + settings.LibreURL + settings.Model + "segmenter-v3-prompt-v1"))
	cacheKey := hex.EncodeToString(hash[:])
	if cached, ok := s.cache[cacheKey]; ok {
		cached.Cached = true
		s.mu.Unlock()
		if err := ctx.Err(); err != nil {
			return Result{}, err
		}
		return cached, nil
	}
	store := s.store
	s.mu.Unlock()
	if store != nil {
		if data, err := store.Get("translation", cacheKey); err == nil && len(data) > 0 {
			var result Result
			if json.Unmarshal(data, &result) == nil && result.Revision == revision {
				result.Cached = true
				if err := ctx.Err(); err != nil {
					return Result{}, err
				}
				return result, nil
			}
		}
	}
	translated, err := s.client.Translate(ctx, kit.Request{Config: config(settings), APIKey: key, Text: body, Revision: revision}, func(update kit.Progress) {
		if len(progress) > 0 && progress[0] != nil {
			progress[0](Progress(update))
		}
	})
	var responseError *kit.AIResponseError
	if errors.As(err, &responseError) {
		err = &AIResponseError{message: responseError.Error(), Diagnostic: ResponseDiagnostic(responseError.Diagnostic)}
	}
	result := Result(translated)
	if err != nil {
		return Result{}, err
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	if store != nil {
		if data, err := json.Marshal(result); err == nil {
			_ = store.Put("translation", cacheKey, data)
		}
	}
	s.mu.Lock()
	if len(s.cache) >= 256 {
		s.cache = map[string]Result{}
	}
	s.cache[cacheKey] = result
	s.mu.Unlock()
	return result, nil
}

func (s *Service) TestConnection(ctx context.Context) error {
	s.mu.Lock()
	settings := s.settings
	key := ""
	var err error
	if keyName := providerKeyName(settings.Provider); keyName != "" {
		key, err = s.keys.Get(keyName)
	}
	s.mu.Unlock()
	if err != nil {
		return err
	}
	err = s.client.TestConnection(ctx, config(settings), key)
	return err
}

func providerKeyName(provider string) string {
	switch provider {
	case "google-cloud":
		return "google"
	case "ai", "libretranslate":
		return provider
	default:
		return ""
	}
}

func (s *Service) SetCache(store *cache.Store) { s.mu.Lock(); defer s.mu.Unlock(); s.store = store }

func (s *Service) ClearCache() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache = map[string]Result{}
	if s.store != nil {
		return s.store.Clear()
	}
	return nil
}
