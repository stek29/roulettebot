package store

import "errors"

type Store interface {
	SetPair(userID, pairID string) error
	GetPair(userID string) (string, error)
	PopPair(userID string) (string, error)
	HasPair(userID string) bool
}

var (
	ErrUserExists   = errors.New("userID/pairID already present")
	ErrUserNotFound = errors.New("userID has no pair")
)
