package staticrouter

import "context"

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
