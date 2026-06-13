package domain

import (
	"errors"
	"strings"
)

type ID struct {
	value string
}

func NewID(value string) (ID, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return ID{}, errors.New("id required")
	}
	return ID{value: value}, nil
}

func MustID(value string) ID {
	id, err := NewID(value)
	if err != nil {
		panic(err)
	}
	return id
}

func (id ID) String() string {
	return id.value
}
