package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteFileAtomicWritesContentAndLeavesNoTempFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.json")

	if err := writeFileAtomic(path, []byte(`{"hello":"world"}`), 0o644); err != nil {
		t.Fatalf("writeFileAtomic: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading written file: %v", err)
	}
	if string(got) != `{"hello":"world"}` {
		t.Errorf("file content = %q, want %q", got, `{"hello":"world"}`)
	}

	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Errorf("expected the .tmp file to be gone after a successful write, stat err = %v", err)
	}
}

func TestWriteFileAtomicOverwritesExistingContent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.json")

	if err := writeFileAtomic(path, []byte("old"), 0o644); err != nil {
		t.Fatalf("first writeFileAtomic: %v", err)
	}
	if err := writeFileAtomic(path, []byte("new"), 0o644); err != nil {
		t.Fatalf("second writeFileAtomic: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading written file: %v", err)
	}
	if string(got) != "new" {
		t.Errorf("file content = %q, want %q (the old content should not linger)", got, "new")
	}
}
