package fluxpicker

import (
	"reflect"
	"sort"
	"testing"

	"github.com/nimble-giant/ailloy/pkg/mold"
)

func TestAggregateSchemasIdentical(t *testing.T) {
	v := mold.FluxVar{Name: "agents.targets", Type: "list", Default: "[claude]"}
	per := map[string][]mold.FluxVar{
		"alpha": {v},
		"beta":  {v},
	}
	unified, conflicts := AggregateSchemas(per)
	if len(unified) != 1 || unified[0].Name != "agents.targets" {
		t.Errorf("unified = %v", unified)
	}
	if len(conflicts) != 0 {
		t.Errorf("expected no conflicts, got %v", conflicts)
	}
}

func TestAggregateSchemasTypeConflict(t *testing.T) {
	per := map[string][]mold.FluxVar{
		"alpha": {{Name: "x", Type: "list"}},
		"beta":  {{Name: "x", Type: "string"}},
	}
	unified, conflicts := AggregateSchemas(per)
	if len(unified) != 1 || unified[0].Name != "x" {
		t.Errorf("unified = %v", unified)
	}
	if got := conflicts["x"]; len(got) != 2 {
		t.Errorf("conflicts[x] = %v, want both molds", got)
	}
}

func TestAggregateSchemasDefaultConflict(t *testing.T) {
	per := map[string][]mold.FluxVar{
		"alpha": {{Name: "theme", Type: "string", Default: "dark"}},
		"beta":  {{Name: "theme", Type: "string", Default: "light"}},
	}
	_, conflicts := AggregateSchemas(per)
	if got := conflicts["theme"]; len(got) != 2 {
		t.Errorf("conflicts[theme] = %v, want both molds", got)
	}
}

func TestAggregateSchemasDisjoint(t *testing.T) {
	per := map[string][]mold.FluxVar{
		"alpha": {{Name: "a", Type: "string"}},
		"beta":  {{Name: "b", Type: "string"}},
	}
	unified, conflicts := AggregateSchemas(per)
	names := []string{}
	for _, v := range unified {
		names = append(names, v.Name)
	}
	sort.Strings(names)
	if !reflect.DeepEqual(names, []string{"a", "b"}) {
		t.Errorf("unified names = %v, want [a b]", names)
	}
	if len(conflicts) != 0 {
		t.Errorf("expected no conflicts, got %v", conflicts)
	}
}

func TestAggregateSchemasEmpty(t *testing.T) {
	unified, conflicts := AggregateSchemas(nil)
	if len(unified) != 0 {
		t.Errorf("unified = %v, want empty", unified)
	}
	if len(conflicts) != 0 {
		t.Errorf("conflicts = %v, want empty", conflicts)
	}
}
