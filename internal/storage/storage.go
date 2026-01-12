package storage

import (
	"context"
	"io"
	"time"
)


type BackupItem struct {
	Key          string
	Size         int64
	LastModified time.Time
}


type Provider interface {

	Upload(ctx context.Context, key string, data io.Reader) error


	Download(ctx context.Context, key string) (io.ReadCloser, error)


	List(ctx context.Context, prefix string) ([]BackupItem, error)
}
