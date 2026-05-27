package json

import (
	"encoding/json"
	"io"

	"github.com/wxdqing/staticrouter/source"
)

type Loader struct{}

func (Loader) SupportedMode() source.Mode {
	return source.ModeJSON
}

func (Loader) Parse(r io.Reader) (*source.Document, error) {
	var doc source.Document
	if err := json.NewDecoder(r).Decode(&doc); err != nil {
		return nil, err
	}
	return &doc, nil
}
