package gobackend

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

const hostResolveStreamActionPrefix = "__spotiflac_host_resolve_stream_v1__:"

type hostResolveStreamActionRequest struct {
	TrackID         string         `json:"track_id"`
	Quality         string         `json:"quality,omitempty"`
	PreparedContext map[string]any `json:"prepared_context,omitempty"`
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

// invokeReservedHostAction handles additive host-only actions before the normal
// extension action dispatcher. Keeping this transport on the existing native
// invokeExtensionAction bridge means Android/iOS bridge code and its lifecycle
// remain unchanged.
func invokeReservedHostAction(extensionID, actionName string) (map[string]any, bool, error) {
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
