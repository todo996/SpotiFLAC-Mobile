package gobackend

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

const hostResolveStreamActionPrefix = "__spotiflac_host_resolve_stream_v1__:"
const hostPrepareStreamActionPrefix = "__spotiflac_host_prepare_stream_v1__:"

type hostResolveStreamActionRequest struct {
	TrackID         string         `json:"track_id"`
	Quality         string         `json:"quality,omitempty"`
	PreparedContext map[string]any `json:"prepared_context,omitempty"`
}

type hostPrepareStreamActionRequest struct {
	SourceProviderID string `json:"source_provider_id"`
	SourceTrackID    string `json:"source_track_id"`
	ISRC             string `json:"isrc,omitempty"`
	TrackName        string `json:"track_name,omitempty"`
	ArtistName       string `json:"artist_name,omitempty"`
	DurationMS       int    `json:"duration_ms,omitempty"`
	DeezerID         string `json:"deezer_id,omitempty"`
}

type hostStreamProviderIDs struct {
	Spotify string
	Deezer  string
	Tidal   string
	Qobuz   string
}

func decodeHostResolveStreamAction(actionName string) (hostResolveStreamActionRequest, bool, error) {
	if !strings.HasPrefix(actionName, hostResolveStreamActionPrefix) {
		return hostResolveStreamActionRequest{}, false, nil
	}

	encoded := strings.TrimPrefix(actionName, hostResolveStreamActionPrefix)
	if encoded == "" {
		return hostResolveStreamActionRequest{}, true, fmt.Errorf("empty host stream request")
	}
	payload, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return hostResolveStreamActionRequest{}, true, fmt.Errorf("invalid host stream request encoding: %w", err)
	}

	var request hostResolveStreamActionRequest
	if err := json.Unmarshal(payload, &request); err != nil {
		return hostResolveStreamActionRequest{}, true, fmt.Errorf("invalid host stream request: %w", err)
	}
	request.TrackID = strings.TrimSpace(request.TrackID)
	request.Quality = strings.TrimSpace(request.Quality)
	if request.TrackID == "" {
		return hostResolveStreamActionRequest{}, true, fmt.Errorf("host stream request is missing track_id")
	}
	return request, true, nil
}

func decodeHostPrepareStreamAction(actionName string) (hostPrepareStreamActionRequest, bool, error) {
	if !strings.HasPrefix(actionName, hostPrepareStreamActionPrefix) {
		return hostPrepareStreamActionRequest{}, false, nil
	}

	encoded := strings.TrimPrefix(actionName, hostPrepareStreamActionPrefix)
	if encoded == "" {
		return hostPrepareStreamActionRequest{}, true, fmt.Errorf("empty host stream preparation request")
	}
	payload, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return hostPrepareStreamActionRequest{}, true, fmt.Errorf("invalid host stream preparation encoding: %w", err)
	}

	var request hostPrepareStreamActionRequest
	if err := json.Unmarshal(payload, &request); err != nil {
		return hostPrepareStreamActionRequest{}, true, fmt.Errorf("invalid host stream preparation request: %w", err)
	}
	request.SourceProviderID = strings.TrimSpace(request.SourceProviderID)
	request.SourceTrackID = strings.TrimSpace(request.SourceTrackID)
	request.ISRC = strings.TrimSpace(request.ISRC)
	request.TrackName = strings.TrimSpace(request.TrackName)
	request.ArtistName = strings.TrimSpace(request.ArtistName)
	request.DeezerID = strings.TrimSpace(request.DeezerID)
	if request.SourceProviderID == "" {
		return hostPrepareStreamActionRequest{}, true, fmt.Errorf("host stream preparation is missing source_provider_id")
	}
	if request.SourceTrackID == "" {
		return hostPrepareStreamActionRequest{}, true, fmt.Errorf("host stream preparation is missing source_track_id")
	}
	if request.DurationMS < 0 {
		request.DurationMS = 0
	}
	return request, true, nil
}

func streamProviderIDsForPreparation(request hostPrepareStreamActionRequest) hostStreamProviderIDs {
	ids := hostStreamProviderIDs{Deezer: request.DeezerID}
	switch strings.ToLower(strings.TrimSpace(request.SourceProviderID)) {
	case "spotify":
		ids.Spotify = request.SourceTrackID
	case "deezer":
		ids.Deezer = request.SourceTrackID
	case "tidal":
		ids.Tidal = request.SourceTrackID
	case "qobuz":
		ids.Qobuz = request.SourceTrackID
	}
	return ids
}

func prepareExtensionStream(extensionID string, request hostPrepareStreamActionRequest) (map[string]any, error) {
	extensionID = strings.TrimSpace(extensionID)
	if extensionID == "" {
		return nil, fmt.Errorf("empty provider ID")
	}

	// An extension-native search result already carries the target provider's
	// own identifier. Avoid a redundant availability call in that case.
	if strings.EqualFold(extensionID, request.SourceProviderID) {
		return map[string]any{
			"success":  true,
			"track_id": request.SourceTrackID,
		}, nil
	}

	ids := streamProviderIDsForPreparation(request)
	trackContext := map[string]any{
		"id":          request.SourceTrackID,
		"provider_id": request.SourceProviderID,
		"name":        request.TrackName,
		"artists":     request.ArtistName,
		"isrc":        request.ISRC,
		"duration_ms": request.DurationMS,
	}
	if ids.Spotify != "" {
		trackContext["spotify_id"] = ids.Spotify
	}
	if ids.Deezer != "" {
		trackContext["deezer_id"] = ids.Deezer
	}
	if ids.Tidal != "" {
		trackContext["tidal_id"] = ids.Tidal
	}
	if ids.Qobuz != "" {
		trackContext["qobuz_id"] = ids.Qobuz
	}

	ext, err := getExtensionManager().GetExtension(extensionID)
	if err != nil {
		return nil, err
	}
	availability, err := newExtensionProviderWrapper(ext).CheckAvailabilityForItemID(
		request.ISRC,
		request.TrackName,
		request.ArtistName,
		ids.Spotify,
		ids.Deezer,
		ids.Tidal,
		ids.Qobuz,
		request.DurationMS,
		"",
		trackContext,
	)
	if err != nil {
		return nil, err
	}
	if availability == nil || !availability.Available {
		reason := "The selected provider could not match this track."
		skipFallback := false
		if availability != nil {
			if strings.TrimSpace(availability.Reason) != "" {
				reason = strings.TrimSpace(availability.Reason)
			}
			skipFallback = availability.SkipFallback
		}
		return map[string]any{
			"success":       false,
			"error_type":    "stream_unavailable",
			"error_message": reason,
			"skip_fallback": skipFallback,
		}, nil
	}

	providerTrackID := strings.TrimSpace(availability.TrackID)
	if providerTrackID == "" {
		return map[string]any{
			"success":       false,
			"error_type":    "missing_provider_track_id",
			"error_message": "The selected provider matched the track but did not return its native track identifier.",
		}, nil
	}

	result := map[string]any{
		"success":  true,
		"track_id": providerTrackID,
	}
	if len(availability.PreparedContext) > 0 {
		result["prepared_context"] = availability.PreparedContext
	}
	return result, nil
}

// invokeReservedHostAction handles additive host-only actions before the normal
// extension action dispatcher. Keeping this transport on the existing native
// invokeExtensionAction bridge means Android/iOS bridge code and its lifecycle
// remain unchanged.
func invokeReservedHostAction(extensionID, actionName string) (map[string]any, bool, error) {
	prepareRequest, prepareHandled, err := decodeHostPrepareStreamAction(actionName)
	if prepareHandled {
		if err != nil {
			return nil, true, err
		}
		result, prepareErr := prepareExtensionStream(extensionID, prepareRequest)
		return result, true, prepareErr
	}

	request, handled, err := decodeHostResolveStreamAction(actionName)
	if !handled || err != nil {
		return nil, handled, err
	}

	preparedJSON := ""
	if len(request.PreparedContext) > 0 {
		raw, marshalErr := json.Marshal(request.PreparedContext)
		if marshalErr != nil {
			return nil, true, fmt.Errorf("failed to encode prepared stream context: %w", marshalErr)
		}
		preparedJSON = string(raw)
	}

	resultJSON, err := ResolveExtensionStreamJSON(
		extensionID,
		request.TrackID,
		request.Quality,
		preparedJSON,
	)
	if err != nil {
		return nil, true, err
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(resultJSON), &result); err != nil {
		return nil, true, fmt.Errorf("failed to decode stream result: %w", err)
	}
	return result, true, nil
}
