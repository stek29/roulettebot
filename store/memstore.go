package store

import "sync"

// MemStore implements in-memory store
type MemStore struct {
	m  map[string]string
	mu sync.RWMutex
}

func NewMemStore() *MemStore {
	return &MemStore{
		m: make(map[string]string),
	}
}

func (m *MemStore) SetPair(userID, pairID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.m[userID]; ok {
		return ErrUserExists
	}
	if _, ok := m.m[pairID]; ok {
		return ErrUserExists
	}

	m.m[userID] = pairID
	m.m[pairID] = userID

	return nil
}

func (m *MemStore) GetPair(userID string) (string, error) {
	m.mu.RLock()
	pairID, ok := m.m[userID]
	m.mu.RUnlock()

	if !ok {
		return "", ErrUserNotFound
	}

	return pairID, nil
}

func (m *MemStore) PopPair(userID string) (string, error) {
	m.mu.Lock()
	pairID, ok := m.m[userID]
	delete(m.m, userID)
	m.mu.Unlock()

	if !ok {
		return "", ErrUserNotFound
	}

	return pairID, nil
}

func (m *MemStore) HasPair(userID string) bool {
	m.mu.RLock()
	_, ok := m.m[userID]
	m.mu.RUnlock()
	return ok
}
