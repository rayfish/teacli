package cmd

import (
	"strings"
	"testing"
)

func TestBuildSchemaIncludesPRList(t *testing.T) {
	schema := buildSchema(RootCmd)
	found := false
	for _, c := range schema.Commands {
		if c.Path == "pr list" {
			found = true
			hasFormat := false
			for _, f := range c.Flags {
				if f.Name == "format" {
					hasFormat = true
				}
			}
			if !hasFormat {
				t.Error("pr list schema missing inherited --format flag")
			}
		}
	}
	if !found {
		t.Error("schema does not include 'pr list'")
	}
	if !strings.HasPrefix(schema.Name, "teacli") {
		t.Errorf("schema.Name = %q, want teacli", schema.Name)
	}
}
