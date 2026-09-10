package gobackend

import (
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/dop251/goja"
)

// invokeCompatibleStreamResolver keeps the new resolveStream extension API as
// the primary contract while providing a narrowly-scoped compatibility bridge
// for the current official providers. The bridge only invokes provider-global
// URL resolution helpers; it never calls download(), writes files, or changes
// the existing download pipeline.
func invokeCompatibleStreamResolver(
	vm *goja.Runtime,
	providerID, trackID, quality string,
	options map[string]any,
) (goja.Value, error) {
	if hasExtensionMethod(vm, "resolveStream") {
		return invokeExtensionMethod(vm, "resolveStream", trackID, quality, options)
	}

	switch strings.ToLower(strings.TrimSpace(providerID)) {
	case "qobuz-web":
		return invokeLegacyQobuzStreamResolver(vm, trackID, quality)
	case "tidal-web":
		return invokeLegacyTidalStreamResolver(vm, trackID, quality)
	case "amazon":
		return legacyStreamFailure(
			vm,
			"stream_requires_processing",
			"Amazon legacy streams may require decryption or container processing; this provider must implement resolveStream for safe playback.",
		), nil
	default:
		return goja.Null(), nil
	}
}

func invokeLegacyQobuzStreamResolver(vm *goja.Runtime, trackID, quality string) (goja.Value, error) {
	value, err := invokeExtensionOrGlobal(
		vm,
		"resolveDownloadInfo",
		trackID,
		quality,
		map[string]any{},
	)
	if err != nil {
		return nil, err
	}
	if gojaValueIsEmpty(value) {
		return legacyStreamFailure(
			vm,
			"not_implemented",
			"Qobuz legacy stream resolver is unavailable.",
		), nil
	}

	directURL := gojaObjectString(vm, value, "directURL", "url")
	if directURL == "" {
		return legacyStreamFailure(
			vm,
			"stream_unavailable",
			"Qobuz did not return a direct audio URL.",
		), nil
	}

	headers, err := legacyUserAgentHeaders(vm, directURL)
	if err != nil {
		return nil, err
	}
	return vm.ToValue(map[string]any{
		"success":      true,
		"url":          directURL,
		"headers":      headers,
		"content_type": "audio/flac",
		"provider":     "qobuz-web",
		"quality":      strings.TrimSpace(quality),
	}), nil
}

func invokeLegacyTidalStreamResolver(vm *goja.Runtime, trackID, quality string) (goja.Value, error) {
	value, err := invokeExtensionOrGlobal(
		vm,
		"fetchDownloadInfo",
		trackID,
		quality,
		map[string]any{},
	)
	if err != nil {
		return nil, err
	}
	if gojaValueIsEmpty(value) {
		return legacyStreamFailure(
			vm,
			"not_implemented",
			"TIDAL legacy stream resolver is unavailable.",
		), nil
	}

	kind := strings.ToLower(gojaObjectString(vm, value, "kind"))
	directURL := gojaObjectString(vm, value, "directURL", "url")
	if kind != "direct" || directURL == "" {
		return legacyStreamFailure(
			vm,
			"stream_format_unsupported",
			"TIDAL returned a segmented or otherwise non-direct stream. The provider must expose a direct URL through resolveStream for playback.",
		), nil
	}

	headers, err := legacyUserAgentHeaders(vm)
	if err != nil {
		return nil, err
	}
	contentType := gojaObjectString(vm, value, "manifestMimeType", "mimeType", "contentType")
	if contentType == "" {
		contentType = inferLegacyStreamContentType(directURL)
	}
	return vm.ToValue(map[string]any{
		"success":      true,
		"url":          directURL,
		"headers":      headers,
		"content_type": contentType,
		"provider":     "tidal-web",
		"quality":      strings.TrimSpace(quality),
	}), nil
}

func legacyUserAgentHeaders(vm *goja.Runtime, args ...any) (map[string]string, error) {
	value, err := invokeExtensionOrGlobal(vm, "requestUserAgent", args...)
	if err != nil {
		return nil, err
	}
	if gojaValueIsEmpty(value) {
		return map[string]string{}, nil
	}
	userAgent := strings.TrimSpace(value.String())
	if userAgent == "" {
		return map[string]string{}, nil
	}
	if strings.ContainsAny(userAgent, "\r\n") {
		return nil, fmt.Errorf("legacy stream resolver returned an invalid User-Agent")
	}
	return map[string]string{"User-Agent": userAgent}, nil
}

func gojaObjectString(vm *goja.Runtime, value goja.Value, keys ...string) string {
	if gojaValueIsEmpty(value) {
		return ""
	}
	object := value.ToObject(vm)
	for _, key := range keys {
		field := object.Get(key)
		if gojaValueIsEmpty(field) {
			continue
		}
		text := strings.TrimSpace(field.String())
		if text != "" {
			return text
		}
	}
	return ""
}

func legacyStreamFailure(vm *goja.Runtime, errorType, message string) goja.Value {
	return vm.ToValue(map[string]any{
		"success":       false,
		"error_type":    errorType,
		"error_message": message,
	})
}

func inferLegacyStreamContentType(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	ext := strings.ToLower(path.Ext(parsed.Path))
	switch ext {
	case ".flac":
		return "audio/flac"
	case ".m4a", ".mp4":
		return "audio/mp4"
	case ".mp3":
		return "audio/mpeg"
	case ".aac":
		return "audio/aac"
	case ".ogg", ".oga":
		return "audio/ogg"
	case ".opus":
		return "audio/ogg; codecs=opus"
	default:
		return ""
	}
}
