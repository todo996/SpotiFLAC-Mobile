package gobackend

import (
	"fmt"
	"sort"
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

func settingConfigured(settings map[string]any, setting ExtensionSetting) bool {
	value, exists := settings[setting.Key]
	if exists {
		if text, ok := value.(string); ok {
			return strings.TrimSpace(text) != ""
		}
		return value != nil
	}
	if setting.Default == nil {
		return false
	}
	if text, ok := setting.Default.(string); ok {
		return strings.TrimSpace(text) != ""
	}
	return true
}

func manifestRequiresRuntimeFeature(manifest *ExtensionManifest, feature string) bool {
	if manifest == nil {
		return false
	}
	for _, raw := range manifest.RequiredRuntimeFeatures {
		name := strings.TrimSpace(raw)
		if at := strings.LastIndex(name, "@"); at > 0 {
			name = name[:at]
		}
		if strings.EqualFold(name, feature) {
			return true
		}
	}
	return false
}

func webProviderStreamMode(ext *loadedExtension) string {
	if ext == nil || ext.Manifest == nil || !ext.Manifest.IsDownloadProvider() {
		return "none"
	}
	providerID := strings.ToLower(strings.TrimSpace(ext.ID))
	switch providerID {
	case "qobuz-web", "tidal-web":
		return "supported"
	case "amazon":
		return "requires_processing"
	}
	for _, key := range []string{"streaming", "resolveStream", "resolve_stream"} {
		if value, ok := ext.Manifest.Capabilities[key].(bool); ok && value {
			return "supported"
		}
	}
	return "extension_defined"
}

// GetWebProviderInventoryJSON returns only non-sensitive provider metadata for
// the Web UI. Secret setting values, access tokens, refresh tokens and pending
// OAuth state are intentionally never serialized here.
func GetWebProviderInventoryJSON() (string, error) {
	extensions := getExtensionManager().GetAllExtensions()
	sort.SliceStable(extensions, func(i, j int) bool {
		left := strings.ToLower(strings.TrimSpace(extensions[i].Manifest.DisplayName))
		right := strings.ToLower(strings.TrimSpace(extensions[j].Manifest.DisplayName))
		if left == right {
			return extensions[i].ID < extensions[j].ID
		}
		return left < right
	})

	providers := make([]map[string]any, 0, len(extensions))
	settingsStore := GetExtensionSettingsStore()
	for _, ext := range extensions {
		if ext == nil || ext.Manifest == nil {
			continue
		}
		manifest := ext.Manifest
		if !manifest.IsMetadataProvider() && !manifest.IsDownloadProvider() && !manifest.IsLyricsProvider() {
			continue
		}

		storedSettings := settingsStore.GetAll(ext.ID)
		missingRequired := make([]string, 0)
		safeSettings := make([]map[string]any, 0, len(manifest.Settings))
		for _, setting := range manifest.Settings {
			configured := setting.Type == SettingTypeButton || settingConfigured(storedSettings, setting)
			label := strings.TrimSpace(setting.Label)
			if label == "" {
				label = setting.Key
			}
			if setting.Required && setting.Type != SettingTypeButton && !configured {
				missingRequired = append(missingRequired, label)
			}
			entry := map[string]any{
				"key":        setting.Key,
				"label":      label,
				"type":       setting.Type,
				"required":   setting.Required,
				"secret":     setting.Secret,
				"configured": configured,
			}
			if len(setting.Options) > 0 {
				entry["options"] = setting.Options
			}
			if setting.Type == SettingTypeButton {
				entry["actionAvailable"] = strings.TrimSpace(setting.Action) != ""
			}
			safeSettings = append(safeSettings, entry)
		}

		pendingAuth := GetPendingAuthRequest(ext.ID) != nil
		authRequired := manifest.SignedSession != nil || manifestRequiresRuntimeFeature(manifest, "webviewAuth") || pendingAuth
		authenticated := !authRequired || IsExtensionAuthenticatedByID(ext.ID)

		status := "ready"
		switch {
		case ext.Error != "":
			status = "error"
		case !ext.Enabled:
			status = "disabled"
		case len(missingRequired) > 0:
			status = "needs_configuration"
		case pendingAuth:
			status = "verification_pending"
		}

		item := map[string]any{
			"id":          ext.ID,
			"displayName": manifest.DisplayName,
			"version":     manifest.Version,
			"enabled":     ext.Enabled,
			"status":      status,
			"capabilities": map[string]any{
				"metadata":   manifest.IsMetadataProvider(),
				"download":   manifest.IsDownloadProvider(),
				"lyrics":     manifest.IsLyricsProvider(),
				"search":     ext.Enabled && ext.Error == "" && manifest.IsMetadataProvider(),
				"streamMode": webProviderStreamMode(ext),
			},
			"configuration": map[string]any{
				"complete":        len(missingRequired) == 0,
				"missingRequired": missingRequired,
				"settings":        safeSettings,
			},
			"auth": map[string]any{
				"required":      authRequired,
				"authenticated": authenticated,
				"pending":       pendingAuth,
			},
		}
		if ext.Error != "" {
			item["error"] = ext.Error
		}
		providers = append(providers, item)
	}

	return marshalJSONString(map[string]any{
		"providers": providers,
		"count":     len(providers),
	})
}
