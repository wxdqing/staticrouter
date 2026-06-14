package source

type Mode string

const (
	ModeXML  Mode = "xml"
	ModeJSON Mode = "json"
)

type Document struct {
	Version int64   `xml:"version,attr" json:"version"`
	Scope   string  `xml:"scope,attr" json:"scope"`
	Routes  []Route `xml:"route" json:"routes"`
}

type Route struct {
	Kinds Kinds `xml:"kinds" json:"kinds"`
	Nodes Nodes `xml:"nodes" json:"nodes"`
}

type Kinds struct {
	Kinds []string `xml:"kind" json:"kind"`
}

type Nodes struct {
	Nodes []Node `xml:"node" json:"node"`
}

type Node struct {
	NodeID    string    `xml:"node_id,attr" json:"node_id"`
	Type      string    `xml:"type,attr" json:"type"`
	RouteKeys RouteKeys `xml:"route_keys" json:"route_keys"`
}

type RouteKeys struct {
	Field  string `xml:"field,attr" json:"field"`
	Keys   Keys   `xml:"keys" json:"keys"`
	Ranges Ranges `xml:"ranges" json:"ranges"`
}

type Keys struct {
	Keys []int32 `xml:"key" json:"key"`
}

type Ranges struct {
	Ranges []Range `xml:"range" json:"range"`
}

type Range struct {
	Start int32 `xml:"start,attr" json:"start"`
	End   int32 `xml:"end,attr" json:"end"`
}
