package database

import (
	"reflect"
	"testing"

	"github.com/vdparikh/make-mcp/backend/internal/models"
)

func TestUniqueValidServerUUIDs(t *testing.T) {
	t.Parallel()
	u1 := "550e8400-e29b-41d4-a716-446655440001"
	u2 := "550e8400-e29b-41d4-a716-446655440002"
	tt := []struct {
		name string
		in   []string
		want []string
	}{
		{name: "empty", in: nil, want: nil},
		{name: "dedupe_and_order", in: []string{u2, u1, u2, " "}, want: []string{u2, u1}},
		{name: "skip_invalid", in: []string{"not-a-uuid", u1}, want: []string{u1}},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := uniqueValidServerUUIDs(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %#v want %#v", got, tc.want)
			}
		})
	}
}

func TestCloneServerShallow(t *testing.T) {
	t.Parallel()
	orig := &models.Server{
		ID:        "550e8400-e29b-41d4-a716-446655440000",
		Name:      "s",
		Tools:     []models.Tool{{ID: "t1", Name: "tool-a"}},
		Resources: []models.Resource{{ID: "r1", Name: "res-a"}},
		Prompts:   []models.Prompt{{ID: "p1", Name: "pr-a"}},
	}
	a := cloneServerShallow(orig)
	b := cloneServerShallow(orig)
	if a == nil || b == nil {
		t.Fatal("expected non-nil clones")
	}
	if a == orig || b == orig {
		t.Fatal("clone must not reuse original pointer")
	}
	a.Tools[0].Name = "mutated"
	if b.Tools[0].Name != "tool-a" {
		t.Fatalf("b.Tools mutated with a: got %q", b.Tools[0].Name)
	}
	if orig.Tools[0].Name != "tool-a" {
		t.Fatalf("original Tools should be unchanged: got %q", orig.Tools[0].Name)
	}
}

func TestCloneServerShallowNil(t *testing.T) {
	t.Parallel()
	if cloneServerShallow(nil) != nil {
		t.Fatal("expected nil")
	}
}
