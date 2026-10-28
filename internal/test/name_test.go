package test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/sinueh/crudding/internal/project"
)

func TestReadProjectName(t *testing.T) {
	var output bytes.Buffer
	name, err := project.ReadProjectName(strings.NewReader("Minha API\n"), &output)
	if err != nil || name != "Minha API" {
		t.Fatalf("ReadProjectName() = %q, %v", name, err)
	}
}

func TestProjectNameValidationAndSlug(t *testing.T) {
	for _, name := range []string{"", "../escape", "bad/name", "bad\\name", " name", "name "} {
		if _, err := project.ReadProjectName(strings.NewReader(name+"\n"), &bytes.Buffer{}); err == nil {
			t.Errorf("ReadProjectName(%q) should fail", name)
		}
	}
	if got := project.Slug("Minha API_2"); got != "minha-api-2" {
		t.Fatalf("Slug() = %q", got)
	}
}

func TestReadEndpointName(t *testing.T) {
	for _, name := range []string{"invoice", "order_item", "v2"} {
		got, err := project.ReadEndpointName(strings.NewReader(name+"\n"), &bytes.Buffer{})
		if err != nil || got != name {
			t.Errorf("ReadEndpointName(%q) = %q, %v", name, got, err)
		}
	}
	for _, name := range []string{"", "Invoice", "1item", "order-item", "_x", "x_", "a__b", "../x", "a/b", "a b"} {
		if _, err := project.ReadEndpointName(strings.NewReader(name+"\n"), &bytes.Buffer{}); err == nil {
			t.Errorf("ReadEndpointName(%q) should fail", name)
		}
	}
}
