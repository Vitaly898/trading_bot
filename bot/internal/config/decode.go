package config

import (
	"bytes"
	"fmt"
	"io"

	"gopkg.in/yaml.v3"
)

func decode(raw []byte, dst any) error {
	var root yaml.Node
	if err := yaml.Unmarshal(raw, &root); err != nil {
		return err
	}
	if err := rejectNulls(&root); err != nil {
		return err
	}
	d := yaml.NewDecoder(bytes.NewReader(raw))
	d.KnownFields(true)
	if err := d.Decode(dst); err != nil {
		return err
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		if err != nil {
			return err
		}
		return fmt.Errorf("only one YAML document is allowed")
	}
	return nil
}

func rejectNulls(node *yaml.Node) error {
	if node.Tag == "!!null" {
		return fmt.Errorf("line %d: null configuration values are not allowed", node.Line)
	}
	for _, child := range node.Content {
		if err := rejectNulls(child); err != nil {
			return err
		}
	}
	return nil
}
