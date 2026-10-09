package translation

import (
	"errors"
	"fmt"
	"sync"

	keychain "github.com/keybase/go-keychain"
)

const keychainService = "me.sogud.skillview.translation"

type MacKeychain struct{}

func keyQuery(account string) keychain.Item {
	item := keychain.NewItem()
	item.SetSecClass(keychain.SecClassGenericPassword)
	item.SetService(keychainService)
	item.SetAccount(account)
	return item
}

func (MacKeychain) Get(account string) (string, error) {
	query := keyQuery(account)
	query.SetMatchLimit(keychain.MatchLimitOne)
	query.SetReturnData(true)
	items, err := keychain.QueryItem(query)
	if errors.Is(err, keychain.ErrorItemNotFound) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("unlock the macOS Keychain to read the translation key")
	}
	if len(items) == 0 {
		return "", nil
	}
	return string(items[0].Data), nil
}

func (MacKeychain) Set(account, value string) error {
	item := keyQuery(account)
	item.SetLabel("SkillView translation")
	item.SetData([]byte(value))
	item.SetSynchronizable(keychain.SynchronizableNo)
	item.SetAccessible(keychain.AccessibleWhenUnlockedThisDeviceOnly)
	err := keychain.AddItem(item)
	if errors.Is(err, keychain.ErrorDuplicateItem) {
		attributes := keychain.NewItem()
		attributes.SetData([]byte(value))
		err = keychain.UpdateItem(keyQuery(account), attributes)
	}
	if err != nil {
		return fmt.Errorf("the translation key could not be saved in macOS Keychain")
	}
	return nil
}

func (MacKeychain) Delete(account string) error {
	err := keychain.DeleteItem(keyQuery(account))
	if errors.Is(err, keychain.ErrorItemNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("the translation key could not be removed from macOS Keychain")
	}
	return nil
}

// Fixtures use an isolated in-memory store and never touch the user's Keychain.
type MemoryKeys struct {
	mu     sync.Mutex
	values map[string]string
}

func NewMemoryKeys() *MemoryKeys { return &MemoryKeys{values: map[string]string{}} }
func (m *MemoryKeys) Get(key string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.values[key], nil
}
func (m *MemoryKeys) Set(key, value string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.values[key] = value
	return nil
}
func (m *MemoryKeys) Delete(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.values, key)
	return nil
}
