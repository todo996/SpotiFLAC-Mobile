package gobackend

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/dop251/goja"
)

// ExtStreamResult is the additive contract returned by an optional extension
// resolveStream(trackID, quality, options) implementation. Existing download()
// implementations are intentionally untouched and remain fully compatible.
type ExtStreamResult struct {
	Success      bool              `json:"success"`
	URL          string            `json:"url,omitempty"`
	Headers      map[string]string `json:"headers,omitempty"`
	ContentType  string            `json:"content_type,omitempty"`
	Provider     string            `json:"provider,omitempty"`
	Quality      string            `json:"quality,omitempty"`
	ExpiresAtMS  int64             `json:"expires_at_ms,omitempty"`
	ErrorType    string            `json:"error_type,omitempty"`
	ErrorMessage string            `json:"error_message,omitempty"`
}

func parseExtensionStreamResultValue(value goja.Value) (ExtStreamResult, error) {
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return ExtStreamResult{
			Success:      false,
			ErrorType:    "not_implemented",
			ErrorMessage: "resolveStream returned null",
		}, nil
	}

	// A bare URL is accepted as a convenience for simple extensions.
	if exported, ok := value.Export().(string); ok {
		result := ExtStreamResult{Success: true, URL: strings.TrimSpace(exported)}
		if err := validateExtensionStreamResult(&result); err != nil {
			return ExtStreamResult{}, err
		}
		return result, nil
	}

	raw, err := json.Marshal(value.Export())
	if err != nil {
		return ExtStreamResult{}, fmt.Errorf("failed to encode resolveStream result: %w", err)
	}

	var result ExtStreamResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return ExtStreamResult{}, fmt.Errorf("failed to parse resolveStream result: %w", err)
	}
	if result.URL != "" && !result.Success && result.ErrorType == "" && result.ErrorMessage == "" {
		// Be forgiving for extension authors who return {url, headers} without
		// an explicit success field, while keeping explicit errors authoritative.
		result.Success = true
	}
	if err := validateExtensionStreamResult(&result); err != nil {
		return ExtStreamResult{}, err
	}
	return result, nil
}

func validateExtensionStreamResult(result *ExtStreamResult) error {
	result.URL = strings.TrimSpace(result.URL)
	result.ContentType = strings.TrimSpace(result.ContentType)
	result.Provider = strings.TrimSpace(result.Provider)
	result.Quality = strings.TrimSpace(result.Quality)
	result.ErrorType = strings.TrimSpace(result.ErrorType)
	result.ErrorMessage = strings.TrimSpace(result.ErrorMessage)

	if !result.Success {
		if result.ErrorType == "" {
			result.ErrorType = "stream_unavailable"
		}
		return nil
	}
	if result.URL == "" {
		return fmt.Errorf("resolveStream succeeded without a URL")
	}

	parsed, err := url.Parse(result.URL)
	if err != nil || parsed.Host == "" {
		return fmt.Errorf("resolveStream returned an invalid URL")
	}
	switch strings.ToLower(parsed.Scheme) {
	case "https":
	case "http":
		// HTTP is allowed here only because extension network policy already
		// decides whether an extension may contact an HTTP origin. The native
		// platform may still reject insecure playback according to its policy.
	default:
		return fmt.Errorf("resolveStream returned an unsupported URL scheme")
	}

	if len(result.Headers) > 0 {
		sanitized := make(map[string]string, len(result.Headers))
		for key, value := range result.Headers {
			key = strings.TrimSpace(key)
			if key == "" || strings.ContainsAny(key, "\r\n") || strings.ContainsAny(value, "\r\n") {
				return fmt.Errorf("resolveStream returned an invalid HTTP header")
			}
			sanitized[key] = value
		}
		result.Headers = sanitized
	}

	return nil
}

// ResolveStream invokes the optional resolveStream extension method. It uses
// the shared extension VM because no file-system write or download worker is
// involved. Signed/temporary URLs are deliberately returned to the caller and
// should be resolved again when playback is retried after expiry.
func (p *extensionProviderWrapper) ResolveStream(
	ctx context.Context,
	trackID, quality string,
	preparedContext map[string]any,
) (*ExtStreamResult, error) {
	if !p.extension.Manifest.IsDownloadProvider() {
		return nil, fmt.Errorf("extension '%s' is not a download provider", p.extension.ID)
	}
	if !p.extension.Enabled {
		return nil, fmt.Errorf("extension '%s' is disabled", p.extension.ID)
	}

	options := map[string]any{
		"resolutionTimeoutMs": extensionResolutionTimeout.Milliseconds(),
	}
	if len(preparedContext) > 0 {
		options["preparedContext"] = preparedContext
	}

	return callExtension(p, extCallOpts{
		perfName: "resolveStream",
		context:  ctx,
		timeout:  extensionResolutionTimeout,
		invoke: func(vm *goja.Runtime) (goja.Value, error) {
			if !hasExtensionMethod(vm, "resolveStream") {
				return goja.Null(), nil
			}
			return invokeExtensionMethod(vm, "resolveStream", trackID, quality, options)
		},
	}, func(perf *extensionCallPerf, value goja.Value) (*ExtStreamResult, error) {
		parseStartedAt := time.Now()
		result, err := parseExtensionStreamResultValue(value)
		perf.recordParse(time.Since(parseStartedAt))
		perf.setItems(1)
		if err != nil {
			return nil, err
		}
		if result.Provider == "" {
			result.Provider = p.extension.ID
		}
		return &result, nil
	})
}

// ResolveExtensionStreamJSON is the gomobile-friendly boundary used by the
// Android/iOS bridges. preparedContextJSON is optional and remains opaque to
// the host so extensions can reuse availability data without changing old
// download behavior.
func ResolveExtensionStreamJSON(providerID, trackID, quality, preparedContextJSON string) (string, error) {
	providerID = strings.TrimSpace(providerID)
	trackID = strings.TrimSpace(trackID)
	quality = strings.TrimSpace(quality)
	if providerID == "" {
		return "", fmt.Errorf("empty provider ID")
	}
	if trackID == "" {
		return "", fmt.Errorf("empty track ID")
	}

	var preparedContext map[string]any
	if strings.TrimSpace(preparedContextJSON) != "" {
		if err := json.Unmarshal([]byte(preparedContextJSON), &preparedContext); err != nil {
			return "", fmt.Errorf("invalid prepared stream context: %w", err)
		}
	}

	ext, err := getExtensionManager().GetExtension(providerID)
	if err != nil {
		return "", err
	}
	result, err := newExtensionProviderWrapper(ext).ResolveStream(
		context.Background(),
		trackID,
		quality,
		preparedContext,
	)
	if err != nil {
		return "", err
	}
	return marshalJSONString(result)
}
