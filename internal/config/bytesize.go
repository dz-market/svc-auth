package config

import (
	"fmt"

	"github.com/docker/go-units"
)

type ByteSize int64

func (b *ByteSize) UnmarshalText(text []byte) error {
	n, err := units.RAMInBytes(string(text))
	if err != nil {
		return fmt.Errorf("parse size %q: %w", text, err)
	}

	*b = ByteSize(n)

	return nil
}

func (b ByteSize) KiB() uint32 {
	//nolint:gosec // bounded by the maxsize validation on the config field
	return uint32(b / units.KiB)
}

func (b ByteSize) Bytes() int {
	return int(b)
}
