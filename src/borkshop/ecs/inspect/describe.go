package inspect

import (
	"fmt"
	"io"
	"unicode/utf8"

	"borkshop/ecs"
)

// DescSpec is a convenience constructor for a descriptor spec.
func DescSpec(typ ecs.Type, label string, desc Descriptor) Spec {
	return Spec{typ, label, desc}
}

// Descriptor is a function capable of describing some entity aspect or
// component.
type Descriptor func(ecs.Entity) string

// Spec specifies how to inspect some aspect or component data within an entity
// domain.
type Spec struct {
	// Type contains all of the entity type bits required for a given descriptor.
	Type ecs.Type
	// Label is a textual name for the aspect/component being described.
	Label string
	// Desc is a descriptor function that describes the fixed aspect/component.
	Desc Descriptor
}

// Describer supports writing entity descriptions given a fixed set of Specs.
type Describer struct {
	Specs []Spec

	setup    bool
	width    int
	keyWidth int
	valWidth int
}

const defaultWidth = 60

func (desc *Describer) init() {
	if desc.setup {
		return
	}
	if desc.width == 0 {
		desc.width = defaultWidth
	}
	desc.keyWidth = 5
	for _, sp := range desc.Specs {
		if c := utf8.RuneCountInString(sp.Label); desc.keyWidth < c {
			desc.keyWidth = c
		}
	}
	desc.valWidth = desc.width - desc.keyWidth - 2
	desc.setup = true
}

// Describe writes all relevant descriptor bytes to the given io.Writer for an entity.
func (desc *Describer) Describe(w io.Writer, ent ecs.Entity) {
	desc.init()
	first := true
	writeRow := func(label, val string) {
		if first {
			first = false
		} else {
			io.WriteString(w, "\r\n")
		}
		_, _ = fmt.Fprintf(w, "% *s", desc.keyWidth, label)
		if val != "" {
			_, _ = fmt.Fprintf(w, ": % *v", desc.valWidth, val)
		}
	}

	typ := ent.Type()
	writeRow("Scope", fmt.Sprintf("%p", ent.Scope))
	writeRow("ID", ent.ID.String())
	writeRow("Type", typ.String())
	for _, sp := range desc.Specs {
		if typ.HasAll(sp.Type) {
			if sp.Desc != nil {
				writeRow(sp.Label, sp.Desc(ent))
			} else {
				writeRow(sp.Label, "")
			}
		}
	}
}

// Describe is a convenience for describign an entity with a temporary Describer.
func Describe(w io.Writer, ent ecs.Entity, specs ...Spec) {
	desc := Describer{Specs: specs}
	desc.Describe(w, ent)
}
