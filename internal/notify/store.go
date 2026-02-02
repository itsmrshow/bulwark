package notify

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/yourusername/bulwark/internal/logging"
	"github.com/yourusername/bulwark/internal/state"
)

// Store persists notification settings.
type Store interface {
	Load(ctx context.Context) (Settings, error)
	Save(ctx context.Context, settings Settings) error
	SetHash(ctx context.Context, hash string)
}

// NewStore chooses a store based on available backends.
func NewStore(cfgPath string, stateStore state.Store, logger *logging.Logger) Store {
	if cfgPath != "" {
		return &fileStore{path: cfgPath, logger: logger}
	}
	if stateStore != nil {
		return &dbStore{store: stateStore}
	}
	return &memoryStore{settings: Defaults()}
}

type memoryStore struct {
	settings Settings
}

func (m *memoryStore) Load(ctx context.Context) (Settings, error) {
	return m.settings, nil
}

func (m *memoryStore) Save(ctx context.Context, settings Settings) error {
	m.settings = settings
	return nil
}

func (m *memoryStore) SetHash(ctx context.Context, hash string) {}

type dbStore struct {
	store state.Store
	hash  string
}

func (d *dbStore) Load(ctx context.Context) (Settings, error) {
	value, err := d.store.GetSetting(ctx, settingsKey)
	if err != nil {
		return Defaults(), nil
	}
	if hash, err := d.store.GetSetting(ctx, lastHashKey); err == nil {
		d.hash = hash
	}
	return Decode(value)
}

func (d *dbStore) Save(ctx context.Context, settings Settings) error {
	encoded, err := Encode(settings)
	if err != nil {
		return err
	}
	if err := d.store.SetSetting(ctx, settingsKey, encoded); err != nil {
		return err
	}
	if d.hash != "" {
		_ = d.store.SetSetting(ctx, lastHashKey, d.hash)
	}
	return nil
}

func (d *dbStore) SetHash(ctx context.Context, hash string) {
	d.hash = hash
	if d.store != nil {
		_ = d.store.SetSetting(ctx, lastHashKey, hash)
	}
}

type fileStore struct {
	mu     sync.Mutex
	path   string
	logger *logging.Logger
	hash   string
}

func (f *fileStore) Load(ctx context.Context) (Settings, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	data, err := os.ReadFile(f.path)
	if err != nil {
		return Defaults(), nil
	}

	var wrapper struct {
		Notifications Settings `json:"notifications"`
		LastHash      string   `json:"last_hash"`
	}
	if err := json.Unmarshal(data, &wrapper); err == nil && (wrapper.Notifications != Settings{}) {
		if wrapper.LastHash != "" {
			f.hash = wrapper.LastHash
		}
		return wrapper.Notifications.Normalize(), nil
	}

	return Decode(string(data))
}

func (f *fileStore) Save(ctx context.Context, settings Settings) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	encoded, err := Encode(settings)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(f.path), 0o755); err != nil {
		return err
	}

	payload, err := json.MarshalIndent(map[string]interface{}{
		"notifications": settings,
		"last_hash":     f.hash,
	}, "", "  ")
	if err != nil {
		payload = []byte(encoded)
	}

	return os.WriteFile(f.path, payload, 0o600)
}

func (f *fileStore) SetHash(ctx context.Context, hash string) {
	f.mu.Lock()
	f.hash = hash
	f.mu.Unlock()
}
