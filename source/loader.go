package source

import "io"

type Loader interface {
	SupportedMode() Mode
	Parse(r io.Reader) (*Document, error)
}
