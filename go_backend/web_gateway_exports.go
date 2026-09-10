package gobackend

import (
	"fmt"
	"strings"
)

// SearchExtensionTracksJSON searches all enabled metadata-provider extensions
// using the same priority, de-duplication, and provider stamping as the mobile
// application. It is intentionally small so non-mobile hosts (for example the
// Web gateway) can reuse the extension runtime without duplicating provider
// logic.
func SearchExtensionTracksJSON(query string, limit int) (string, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return "", fmt.Errorf("search query is required")
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}

	tracks, err := getExtensionManager().SearchTracksWithMetadataProviders(query, limit, true)
	if err != nil {
		return "", err
	}

	result := make([]map[string]any, 0, len(tracks))
	for _, track := range tracks {
		result = append(result, normalizeExtensionTrackMetadataMap(track, "", 0))
	}

	return marshalJSONString(map[string]any{
		"tracks":   result,
		"provider": "extensions",
	})
}

// GetExtensionMetadataProviderCount reports how many enabled and healthy
// metadata providers are currently available to the extension manager.
func GetExtensionMetadataProviderCount() int {
	return len(getExtensionManager().GetMetadataProviders())
}
