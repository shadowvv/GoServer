package wordFilter

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadCleansWords(t *testing.T) {
	path := writeTestDict(t, "\xef\xbb\xbfbad \n dirty\n\n")

	wf, err := newWordFilter(path)
	if err != nil {
		t.Fatalf("newWordFilter() error = %v", err)
	}

	if !wf.HasSensitive("bad") {
		t.Fatal("expected BOM and trailing whitespace to be removed from word")
	}
	if !wf.HasSensitive("dirty") {
		t.Fatal("expected leading whitespace to be removed from word")
	}
}

func TestPackageFunctionsWhenNotInitialized(t *testing.T) {
	old := wordFilterService.Load()
	wordFilterService.Store(nil)
	t.Cleanup(func() {
		wordFilterService.Store(old)
	})

	if !HasSensitive("anything") {
		t.Fatal("HasSensitive should fail closed when service is not initialized")
	}
	if err := Reload(); !errors.Is(err, ErrWordFilterNotInitialized) {
		t.Fatalf("Reload() error = %v, want %v", err, ErrWordFilterNotInitialized)
	}
	if got := Find("anything"); got != nil {
		t.Fatalf("Find() = %v, want nil", got)
	}
	if got := Replace("anything"); got != "anything" {
		t.Fatalf("Replace() = %q, want original text", got)
	}
}

func TestFindAndReplaceRemoveNoise(t *testing.T) {
	path := writeTestDict(t, "foobar\n")

	wf, err := newWordFilter(path)
	if err != nil {
		t.Fatalf("newWordFilter() error = %v", err)
	}

	if got := wf.Find("foo bar"); !reflect.DeepEqual(got, []string{"foobar"}) {
		t.Fatalf("Find() = %v, want [foobar]", got)
	}
	if got := wf.Replace("foo bar"); got != "******" {
		t.Fatalf("Replace() = %q, want %q", got, "******")
	}
}

func writeTestDict(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "dirtyWord.txt")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write test dict: %v", err)
	}
	return path
}
