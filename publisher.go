package staticrouter

import (
	"bytes"
	"context"
)

type Publisher struct {
	store SnapshotStore
}

func NewPublisher(store SnapshotStore) *Publisher {
	return &Publisher{store: store}
}

func (p *Publisher) PublishFile(ctx context.Context, path string) error {
	snapshot, err := LoadRouteSnapshotFromFile(path)
	if err != nil {
		return err
	}
	if p.store == nil {
		return nil
	}
	return p.store.ReplaceSnapshot(ctx, snapshot)
}

func (p *Publisher) Publish(ctx context.Context, mode ConfigMode, content []byte) error {
	snapshot, err := LoadRouteSnapshot(mode, bytes.NewReader(content))
	if err != nil {
		return err
	}
	if p.store == nil {
		return nil
	}
	return p.store.ReplaceSnapshot(ctx, snapshot)
}
