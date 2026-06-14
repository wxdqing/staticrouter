package staticrouter

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	jsonformat "github.com/wxdqing/staticrouter/formats/json"
	xmlformat "github.com/wxdqing/staticrouter/formats/xml"
	"github.com/wxdqing/staticrouter/source"
)

type ConfigDocument = source.Document
type ConfigRoute = source.Route
type ConfigKinds = source.Kinds
type ConfigNodes = source.Nodes
type ConfigNode = source.Node
type ConfigRouteKeys = source.RouteKeys
type ConfigKeys = source.Keys
type ConfigRanges = source.Ranges
type ConfigRange = source.Range
type ConfigMode = source.Mode

const (
	ConfigModeXML  = source.ModeXML
	ConfigModeJSON = source.ModeJSON
)

var configLoadersByMode = map[source.Mode]source.Loader{
	source.ModeXML:  xmlformat.Loader{},
	source.ModeJSON: jsonformat.Loader{},
}

var configLoadersByExtension = map[string]source.Mode{
	".xml":  source.ModeXML,
	".json": source.ModeJSON,
}

func Parse(mode ConfigMode, r io.Reader) (*ConfigDocument, error) {
	loader, ok := configLoadersByMode[mode]
	if !ok {
		return nil, fmt.Errorf("staticrouter: unsupported config mode: %s", mode)
	}
	return loader.Parse(r)
}

func LoadRouteSnapshot(mode ConfigMode, r io.Reader) (*RouteSnapshot, error) {
	doc, err := Parse(mode, r)
	if err != nil {
		return nil, err
	}
	return documentToSnapshot(doc)
}

func LoadRouteSnapshotFromFile(path string) (*RouteSnapshot, error) {
	doc, err := loadConfigDocument(path)
	if err != nil {
		return nil, err
	}
	return documentToSnapshot(doc)
}

func loadConfigDocument(path string) (*ConfigDocument, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(path))
	mode, ok := configLoadersByExtension[ext]
	if !ok {
		return nil, fmt.Errorf("staticrouter: unsupported config format: %s", path)
	}
	return Parse(mode, file)
}

func documentToRouteRecords(doc *ConfigDocument) ([]*RouteRecord, error) {
	if doc == nil {
		return nil, nil
	}

	out := make([]*RouteRecord, 0)
	for i := range doc.Routes {
		records, err := routeToRouteRecords(&doc.Routes[i])
		if err != nil {
			return nil, err
		}
		out = append(out, records...)
	}
	return out, nil
}

func documentToSnapshot(doc *ConfigDocument) (*RouteSnapshot, error) {
	records, err := documentToRouteRecords(doc)
	if err != nil {
		return nil, err
	}
	return NormalizeSnapshot(&RouteSnapshot{
		Version: doc.Version,
		Scope:   doc.Scope,
		Routes:  records,
	})
}

func routeToRouteRecords(route *ConfigRoute) ([]*RouteRecord, error) {
	if route == nil {
		return nil, nil
	}
	if len(route.Kinds.Kinds) == 0 {
		return nil, fmt.Errorf("staticrouter: route kinds is empty")
	}
	if len(route.Nodes.Nodes) == 0 {
		return nil, fmt.Errorf("staticrouter: route nodes is empty")
	}

	out := make([]*RouteRecord, 0, len(route.Kinds.Kinds)*len(route.Nodes.Nodes))
	for _, kind := range route.Kinds.Kinds {
		for _, node := range route.Nodes.Nodes {
			nodeType := strings.TrimSpace(node.Type)
			if nodeType == "" {
				nodeType = "game"
			}
			record := &RouteRecord{
				Kind:           kind,
				NodeType:       nodeType,
				RouteKeys:      append([]int32(nil), node.RouteKeys.Keys.Keys...),
				NodeId:         node.NodeID,
				RouteKeyField:  strings.TrimSpace(node.RouteKeys.Field),
			}
			if len(node.RouteKeys.Ranges.Ranges) > 1 {
				return nil, fmt.Errorf("staticrouter: node %s has more than one range", node.NodeID)
			}
			if len(node.RouteKeys.Ranges.Ranges) == 1 {
				record.RouteKeyStart = node.RouteKeys.Ranges.Ranges[0].Start
				record.RouteKeyEnd = node.RouteKeys.Ranges.Ranges[0].End
			}
			out = append(out, record)
		}
	}
	return out, nil
}
