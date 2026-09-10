package gobackend

import (
	"strings"
	"testing"

	"github.com/dop251/goja"
)

func parseCompatibleStreamResult(
	t *testing.T,
	vm *goja.Runtime,
	providerID, trackID, quality string,
) ExtStreamResult {
	t.Helper()
	value, err := invokeCompatibleStreamResolver(
		vm,
		providerID,
		trackID,
		quality,
		map[string]any{"resolutionTimeoutMs": 5000},
	)
	if err != nil {
		t.Fatalf("invokeCompatibleStreamResolver returned error: %v", err)
	}
	result, err := parseExtensionStreamResultValue(value)
	if err != nil {
		t.Fatalf("parseExtensionStreamResultValue returned error: %v", err)
	}
	return result
}

func TestCompatibleStreamResolverPrefersRegisteredResolveStream(t *testing.T) {
	vm := goja.New()
	_, err := vm.RunString(`
		var extension = {
			resolveStream: function(trackID, quality, options) {
				return {
					url: "https://custom.example.test/" + trackID + ".flac",
					provider: "custom",
					quality: quality
				};
			}
		};
		function resolveDownloadInfo() {
			throw new Error("legacy resolver must not run");
		}
	`)
	if err != nil {
		t.Fatal(err)
	}

	result := parseCompatibleStreamResult(t, vm, "qobuz-web", "123", "LOSSLESS")
	if !result.Success || result.Provider != "custom" {
		t.Fatalf("unexpected result: %#v", result)
	}
	if result.URL != "https://custom.example.test/123.flac" {
		t.Fatalf("unexpected URL: %s", result.URL)
	}
}

func TestLegacyQobuzStreamResolverReturnsDirectURLAndUserAgent(t *testing.T) {
	vm := goja.New()
	_, err := vm.RunString(`
		function resolveDownloadInfo(trackID, quality, rejected) {
			if (trackID !== "42") throw new Error("wrong track");
			if (quality !== "HI_RES_LOSSLESS") throw new Error("wrong quality");
			return {
				directURL: "https://streaming-qobuz-std.akamaized.net/audio/42.flac",
				qualityCode: "27"
			};
		}
		function requestUserAgent(url) {
			if (url.indexOf("42.flac") < 0) throw new Error("wrong URL");
			return "SpotiFLAC-Qobuz-Test/1.0";
		}
	`)
	if err != nil {
		t.Fatal(err)
	}

	result := parseCompatibleStreamResult(t, vm, "qobuz-web", "42", "HI_RES_LOSSLESS")
	if !result.Success || result.URL == "" || result.ContentType != "audio/flac" {
		t.Fatalf("unexpected result: %#v", result)
	}
	if result.Headers["User-Agent"] != "SpotiFLAC-Qobuz-Test/1.0" {
		t.Fatalf("missing legacy User-Agent header: %#v", result.Headers)
	}
}

func TestLegacyTidalStreamResolverAcceptsDirectStream(t *testing.T) {
	vm := goja.New()
	_, err := vm.RunString(`
		function fetchDownloadInfo(trackID, quality, rejected) {
			return {
				kind: "direct",
				directURL: "https://audio.tidal.example.test/track-7.m4a",
				manifestMimeType: "audio/mp4"
			};
		}
		function requestUserAgent() {
			return "SpotiFLAC-Tidal-Test/1.0";
		}
	`)
	if err != nil {
		t.Fatal(err)
	}

	result := parseCompatibleStreamResult(t, vm, "tidal-web", "7", "LOSSLESS")
	if !result.Success || result.ContentType != "audio/mp4" {
		t.Fatalf("unexpected result: %#v", result)
	}
	if result.Headers["User-Agent"] != "SpotiFLAC-Tidal-Test/1.0" {
		t.Fatalf("missing legacy User-Agent header: %#v", result.Headers)
	}
}

func TestLegacyTidalStreamResolverRejectsSegmentedStream(t *testing.T) {
	vm := goja.New()
	_, err := vm.RunString(`
		function fetchDownloadInfo() {
			return {
				kind: "segments",
				initURL: "https://audio.example.test/init.mp4",
				mediaURLs: ["https://audio.example.test/one.m4s"]
			};
		}
	`)
	if err != nil {
		t.Fatal(err)
	}

	result := parseCompatibleStreamResult(t, vm, "tidal-web", "7", "LOSSLESS")
	if result.Success || result.ErrorType != "stream_format_unsupported" {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestLegacyAmazonStreamResolverFailsSafely(t *testing.T) {
	vm := goja.New()
	result := parseCompatibleStreamResult(t, vm, "amazon", "asin", "LOSSLESS")
	if result.Success || result.ErrorType != "stream_requires_processing" {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestUnknownLegacyProviderRemainsNotImplemented(t *testing.T) {
	vm := goja.New()
	result := parseCompatibleStreamResult(t, vm, "unknown-provider", "1", "LOSSLESS")
	if result.Success || !strings.EqualFold(result.ErrorType, "not_implemented") {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestInferLegacyStreamContentType(t *testing.T) {
	cases := map[string]string{
		"https://example.test/song.flac?token=1": "audio/flac",
		"https://example.test/song.m4a":          "audio/mp4",
		"https://example.test/song.mp3":          "audio/mpeg",
		"https://example.test/song.bin":          "",
	}
	for input, want := range cases {
		if got := inferLegacyStreamContentType(input); got != want {
			t.Fatalf("inferLegacyStreamContentType(%q) = %q, want %q", input, got, want)
		}
	}
}
