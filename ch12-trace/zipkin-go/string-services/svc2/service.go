//go:build go1.7
// +build go1.7

package svc2

import (
	"context"
	"errors"
)

// Service constants
const (
	Int64Max = 1<<63 - 1
	Int64Min = -(Int64Max + 1)
)

// Service errors
var (
	ErrIntOverflow = errors.New("integer overflow occurred")
)

// Service svc2暴露出来的服务。
type Service interface {
	Sum(ctx context.Context, a int64, b int64) (int64, error)
}
