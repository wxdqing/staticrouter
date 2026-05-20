package staticrouter

import (
	"crypto/md5"
	"encoding/hex"

	"google.golang.org/protobuf/proto"
)

func NormalizeSnapshot(snapshot *RouteSnapshot) (*RouteSnapshot, error) {
	if snapshot == nil {
		return &RouteSnapshot{}, nil
	}

	normalized := proto.Clone(snapshot).(*RouteSnapshot)
	normalized.Checksum = ""
	data, err := proto.Marshal(normalized)
	if err != nil {
		return nil, err
	}
	sum := md5.Sum(data)
	normalized.Checksum = hex.EncodeToString(sum[:])
	return normalized, nil
}
