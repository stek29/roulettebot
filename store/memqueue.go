package store

import "sync"

// MemQueue is randomized user queue, which relies on
// go map range iteration randomness
type MemQueue struct {
	data map[string]struct{}
	mu   sync.RWMutex
}

func NewMemQueue() *MemQueue {
	return &MemQueue{
		data: make(map[string]struct{}),
	}
}

func (m *MemQueue) Has(userID string) bool {
	m.mu.RLock()
	_, has := m.data[userID]
	m.mu.RUnlock()
	return has
}

func (m *MemQueue) Add(userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, has := m.data[userID]; has {
		return ErrUserExists
	}

	m.data[userID] = struct{}{}
	return nil
}

func (m *MemQueue) Remove(userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, has := m.data[userID]; !has {
		return ErrUserNotFound
	}

	delete(m.data, userID)
	return nil
}

func (m *MemQueue) Pick() (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.data) == 0 {
		return "", ErrQueueEmpty
	}

	var userID string
	for userID = range m.data {
		break
	}

	delete(m.data, userID)
	return userID, nil
}
