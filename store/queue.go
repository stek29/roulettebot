package store

import "errors"

type Queue interface {
	Add(userID string) error
	Pick() (userID string, err error)
	Remove(userID string) error
	Has(userID string) bool
}

var (
	ErrQueueEmpty = errors.New("queue is empty")
)
