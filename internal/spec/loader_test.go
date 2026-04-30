package spec

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const minimalSpecYAML = `
openapi: "3.0.3"
info:
  title: Loader Test
  version: "1"
paths: {}
`

// --- readSource ----------------------------------------------------------------------------------


func TestReadSource_File(t *testing.T) {
	path := filepath.Join(t.TempDir(), "spec.yaml")
	if err := os.WriteFile(path, []byte("hello: world"), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	got, err := readSource(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != "hello: world" {
		t.Errorf("got %q, want %q", got, "hello: world")
	}
}

func TestReadSource_FileNotFound(t *testing.T) {
	_, err := readSource("/nonexistent/path/spec.yaml")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "reading spec file") {
		t.Errorf("error %q does not contain %q", err.Error(), "reading spec file")
	}
}

func TestReadSource_HTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("hello: world"))
	}))
	defer srv.Close()

	got, err := readSource(srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != "hello: world" {
		t.Errorf("got %q, want %q", got, "hello: world")
	}
}

func TestReadSource_HTTPError(t *testing.T) {
	_, err := readSource("http://localhost:1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "fetching spec from") {
		t.Errorf("error %q does not contain %q", err.Error(), "fetching spec from")
	}
}


// --- LoadModel -----------------------------------------------------------------------------------


func TestLoadModel_ValidFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "spec.yaml")
	if err := os.WriteFile(path, []byte(minimalSpecYAML), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	model, err := LoadModel(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if model == nil {
		t.Fatal("expected non-nil model")
	}
	if model.Model.Info.Title != "Loader Test" {
		t.Errorf("Title = %q, want %q", model.Model.Info.Title, "Loader Test")
	}
}

func TestLoadModel_FileNotFound(t *testing.T) {
	_, err := LoadModel("/nonexistent/spec.yaml")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestLoadModel_InvalidYAML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "spec.yaml")
	if err := os.WriteFile(path, []byte("{invalid yaml ["), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	_, err := LoadModel(path)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "parsing spec") {
		t.Errorf("error %q does not contain %q", err.Error(), "parsing spec")
	}
}

func TestLoadModel_ValidHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(minimalSpecYAML))
	}))
	defer srv.Close()

	model, err := LoadModel(srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if model == nil {
		t.Fatal("expected non-nil model")
	}
}
