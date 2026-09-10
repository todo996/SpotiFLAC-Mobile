package gobackend

import (
	"strings"
	"testing"

	"github.com/dop251/goja"
)

func TestParseExtensionStreamResultValueAcceptsURLString(t *testing.T) {
	vm := goja.New()
	result, err := parseExtensionStreamResultValue(vm.ToValue("https://cdn.example.com/song.flac"))
	if err != nil {
		t.Fatalf("parseExtensionStreamResultValue returned error: %v", err)
	}
	if !result.Success || result.URL != "https://cdn.example.com/song.flac" {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestParseExtensionStreamResultValueAcceptsObjectAndHeaders(t *testing.T) {
	vm := goja.New()
	value, err := vm.RunString(`({
		url: "https://cdn.example.com/song.flac",
		headers: { Authorization: "Bearer test" },
		content_type: "audio/flac",
		expires_at_ms: 123456789
	})`)
	if err != nil {
		t.Fatal(err)
	}
	result, err := parseExtensionStreamResultValue(value)
	if err != nil {
		t.Fatalf("parseExtensionStreamResultValue returned error: %v", err)
	}
	if !result.Success || result.Headers["Authorization"] != "Bearer test" || result.ContentType != "audio/flac" {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestParseExtensionStreamResultValueRejectsUnsafeURLAndHeader(t *testing.T) {
	vm := goja.New()
	for _, source := range []string{
		`({success:true,url:"file:///tmp/song.flac"})`,
		`({success:true,url:"https://cdn.example.com/song.flac",headers:{"X-Test":"ok\\r\\nInjected: yes"}})`,
	} {
		value, err := vm.RunString(source)
		if err != nil {
			t.Fatal(err)
		}
		_, err = parseExtensionStreamResultValue(value)
		if err == nil {
			t.Fatalf("expected invalid stream result to fail: %s", source)
		}
	}
}

func TestParseExtensionStreamResultValueReturnsNotImplementedForNull(t *testing.T) {
	result, err := parseExtensionStreamResultValue(goja.Null())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success || !strings.EqualFold(result.ErrorType, "not_implemented") {
		t.Fatalf("unexpected result: %#v", result)
	}
}
