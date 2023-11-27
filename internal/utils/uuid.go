package utils

import (
	"strings"

	"github.com/google/uuid"
)

func NewUuidWoDashes() string {
	return strings.Replace(uuid.New().String(), "-", "", -1)
}
