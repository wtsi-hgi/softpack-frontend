package compressed

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFile(t *testing.T) {
	var f File

	s := httptest.NewServer(&f)

	const testData = "MY DATA"

	f.Encode(testData)

	if out := readData(t, s.URL); out != testData {
		t.Fatalf("expected to read %q, got %q", testData, out)
	}

	oldTime := f.modTime

	f.Encode(testData)

	if !f.modTime.Equal(oldTime) {
		t.Fatalf("expecting modtime of %s, got %s", oldTime, f.modTime)
	}

	const newData = "NEW DATA"

	f.Encode(newData)

	if f.modTime.Equal(oldTime) {
		t.Fatalf("expecting modtime to not be %s", oldTime)
	}

	if out := readData(t, s.URL); out != newData {
		t.Fatalf("expected to read %q, got %q", newData, out)
	}
}

func readData(t *testing.T, url string) string {
	t.Helper()

	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("unexpected error getting data: %s", err)
	}

	var out string

	if err = json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("unexpected error decoding data: %s", err)
	}

	return out
}
