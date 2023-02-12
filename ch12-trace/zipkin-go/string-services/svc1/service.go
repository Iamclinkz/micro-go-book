//go:build go1.7
// +build go1.7

package svc1

import (
	"context"
	"errors"
)

// Service constants
const (
	StrMaxSize = 1024
)

// Service errors
var (
	ErrMaxSize = errors.New("maximum size of 1024 bytes exceeded")
)

// Service interface
//svc1暴露出来的服务
type Service interface {
	Concat(ctx context.Context, a, b string) (string, error)
	Sum(ctx context.Context, a, b int64) (int64, error)
}
