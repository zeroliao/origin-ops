package authn

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"sync"
	"time"
)

const sessionTokenBytes = 32

type session struct {
	principal Principal
	expiresAt time.Time
}

type Manager struct {
	store    *Store
	lifetime time.Duration
	now      func() time.Time
	mu       sync.Mutex
	sessions map[[sha256.Size]byte]session
}

func NewManager(store *Store, lifetime time.Duration) *Manager {
	return &Manager{
		store: store, lifetime: lifetime, now: time.Now,
		sessions: make(map[[sha256.Size]byte]session),
	}
}

func (m *Manager) Login(username, password string) (string, Principal, time.Time, bool, error) {
	principal, valid, err := m.store.Authenticate(username, password)
	if err != nil || !valid {
		return "", Principal{}, time.Time{}, valid, err
	}
	tokenBytes := make([]byte, sessionTokenBytes)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", Principal{}, time.Time{}, false, err
	}
	expiresAt := m.now().UTC().Add(m.lifetime)
	tokenHash := sha256.Sum256(tokenBytes)
	m.mu.Lock()
	m.pruneLocked()
	m.sessions[tokenHash] = session{principal: principal, expiresAt: expiresAt}
	m.mu.Unlock()
	return base64.RawURLEncoding.EncodeToString(tokenBytes), principal, expiresAt, true, nil
}

func (m *Manager) Current(token string) (Principal, bool, error) {
	tokenBytes, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(tokenBytes) != sessionTokenBytes {
		return Principal{}, false, nil
	}
	tokenHash := sha256.Sum256(tokenBytes)
	m.mu.Lock()
	m.pruneLocked()
	stored, exists := m.sessions[tokenHash]
	m.mu.Unlock()
	if !exists {
		return Principal{}, false, nil
	}
	valid, err := m.store.Validate(stored.principal)
	if err != nil {
		return Principal{}, false, err
	}
	if !valid {
		m.mu.Lock()
		delete(m.sessions, tokenHash)
		m.mu.Unlock()
		return Principal{}, false, nil
	}
	return stored.principal, true, nil
}

func (m *Manager) Logout(token string) {
	tokenBytes, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(tokenBytes) != sessionTokenBytes {
		return
	}
	tokenHash := sha256.Sum256(tokenBytes)
	m.mu.Lock()
	delete(m.sessions, tokenHash)
	m.mu.Unlock()
}

func (m *Manager) pruneLocked() {
	now := m.now().UTC()
	for token, stored := range m.sessions {
		if !stored.expiresAt.After(now) {
			delete(m.sessions, token)
		}
	}
}
