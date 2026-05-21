package xml

import (
	"encoding/xml"
	"io"

	"staticrouter/source"
)

type Loader struct{}

func (Loader) SupportedMode() source.Mode {
	return source.ModeXML
}

func (Loader) Parse(r io.Reader) (*source.Document, error) {
	var doc source.Document
	if err := xml.NewDecoder(r).Decode(&doc); err != nil {
		return nil, err
	}
	return &doc, nil
}
