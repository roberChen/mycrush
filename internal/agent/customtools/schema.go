package customtools

import (
	"maps"
	"slices"
)

// ParamSchema is the JSON-schema fragment advertised to the model for a
// single input parameter. It is deliberately a map[string]any so this
// package stays free of the fantasy dependency.
type ParamSchema = map[string]any

// BuildParameters turns the definition's input parameters into a
// JSON-schema-style property map and a required-name list, matching the
// shape that fantasy produces for built-in tools (see schema.ToParameters).
// The returned properties map is map[string]any so it can be assigned
// directly to fantasy.ToolInfo.Parameters; each property value is itself a
// map[string]any describing that parameter. Defaults are injected into the
// schema so the model is aware of them.
func (d *Definition) BuildParameters() (properties map[string]any, required []string) {
	params := d.EffectiveParams()
	properties = make(map[string]any, len(params))
	for _, p := range params {
		schema := ParamSchema{
			"type": paramType(p),
		}
		if p.Description != "" {
			schema["description"] = p.Description
		}
		if len(p.Enum) > 0 {
			schema["enum"] = p.Enum
		}
		if p.Default != nil {
			schema["default"] = p.Default
		}
		properties[p.Name] = schema
		if p.Required {
			required = append(required, p.Name)
		}
	}
	return properties, required
}

// paramType normalizes a Param.Type to a JSON-schema type, defaulting to
// "string".
func paramType(p Param) string {
	if p.Type == "" {
		return "string"
	}
	return p.Type
}

// ParamNames returns the ordered list of parameter names (after applying the
// default "prompt" fallback).
func (d *Definition) ParamNames() []string {
	params := d.EffectiveParams()
	names := make([]string, 0, len(params))
	for _, p := range params {
		names = append(names, p.Name)
	}
	return names
}

// Clone returns a deep copy of the definition. The mutable slices/maps are
// copied so callers can adjust a definition without affecting the original.
func (d *Definition) Clone() *Definition {
	if d == nil {
		return nil
	}
	clone := *d
	clone.AllowedTools = slices.Clone(d.AllowedTools)
	clone.Skills = slices.Clone(d.Skills)
	clone.Params = slices.Clone(d.Params)
	clone.AllowedMCP = maps.Clone(d.AllowedMCP)
	return &clone
}
