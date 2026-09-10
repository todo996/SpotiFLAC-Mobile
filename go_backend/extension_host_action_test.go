package gobackend

import (
	"encoding/base64"
	"testing"
)

func TestDecodeHostResolveStreamActionIgnoresNormalActions(t *testing.T) {
	_, handled, err := decodeHostResolveStreamAction("completeGrant")
	if err != nil || handled {
		t.Fatalf("normal action should not be handled: handled=%v err=%v", handled, err)
	}
}

func TestDecodeHostResolveStreamActionParsesPayload(t *testing.T) {
	payload := `{"track_id":"track-123","quality":"lossless","prepared_context":{"token":"abc"}}`
	action := hostResolveStreamActionPrefix + base64.RawURLEncoding.EncodeToString([]byte(payload))
	request, handled, err := decodeHostResolveStreamAction(action)
	if err != nil {
		t.Fatalf("decodeHostResolveStreamAction returned error: %v", err)
	}
	if !handled || request.TrackID != "track-123" || request.Quality != "lossless" {
		t.Fatalf("unexpected request: handled=%v request=%#v", handled, request)
	}
	if request.PreparedContext["token"] != "abc" {
		t.Fatalf("prepared context not preserved: %#v", request.PreparedContext)
	}
}

func TestDecodeHostResolveStreamActionRejectsMissingTrack(t *testing.T) {
	payload := `{"quality":"lossless"}`
	action := hostResolveStreamActionPrefix + base64.RawURLEncoding.EncodeToString([]byte(payload))
	_, handled, err := decodeHostResolveStreamAction(action)
	if !handled || err == nil {
		t.Fatalf("expected missing track_id error: handled=%v err=%v", handled, err)
	}
}

func TestDecodeHostPrepareStreamActionParsesMatchMetadata(t *testing.T) {
	payload := `{"source_provider_id":"spotify","source_track_id":"sp-123","isrc":"USABC1234567","track_name":"Example Song","artist_name":"Example Artist","duration_ms":201000,"deezer_id":"dz-9"}`
	action := hostPrepareStreamActionPrefix + base64.RawURLEncoding.EncodeToString([]byte(payload))
	request, handled, err := decodeHostPrepareStreamAction(action)
	if err != nil {
		t.Fatalf("decodeHostPrepareStreamAction returned error: %v", err)
	}
	if !handled {
		t.Fatal("prepare action should be handled")
	}
	if request.SourceProviderID != "spotify" || request.SourceTrackID != "sp-123" {
		t.Fatalf("unexpected source identity: %#v", request)
	}
	if request.ISRC != "USABC1234567" || request.DurationMS != 201000 || request.DeezerID != "dz-9" {
		t.Fatalf("match metadata not preserved: %#v", request)
	}
}

func TestDecodeHostPrepareStreamActionRejectsMissingSourceIdentity(t *testing.T) {
	payload := `{"source_provider_id":"spotify","track_name":"Example Song"}`
	action := hostPrepareStreamActionPrefix + base64.RawURLEncoding.EncodeToString([]byte(payload))
	_, handled, err := decodeHostPrepareStreamAction(action)
	if !handled || err == nil {
		t.Fatalf("expected missing source_track_id error: handled=%v err=%v", handled, err)
	}
}

func TestStreamProviderIDsForPreparationUsesKnownSourceAndExplicitDeezerID(t *testing.T) {
	ids := streamProviderIDsForPreparation(hostPrepareStreamActionRequest{
		SourceProviderID: "spotify",
		SourceTrackID:    "sp-123",
		DeezerID:         "dz-456",
	})
	if ids.Spotify != "sp-123" || ids.Deezer != "dz-456" || ids.Tidal != "" || ids.Qobuz != "" {
		t.Fatalf("unexpected provider ids: %#v", ids)
	}

	ids = streamProviderIDsForPreparation(hostPrepareStreamActionRequest{
		SourceProviderID: "QOBUZ",
		SourceTrackID:    "qb-789",
	})
	if ids.Qobuz != "qb-789" {
		t.Fatalf("expected case-insensitive Qobuz source mapping: %#v", ids)
	}
}

func TestPrepareExtensionStreamKeepsSameProviderNativeID(t *testing.T) {
	result, err := prepareExtensionStream("provider-a", hostPrepareStreamActionRequest{
		SourceProviderID: "PROVIDER-A",
		SourceTrackID:    "native-123",
	})
	if err != nil {
		t.Fatalf("same-provider preparation returned error: %v", err)
	}
	if result["success"] != true || result["track_id"] != "native-123" {
		t.Fatalf("unexpected same-provider preparation result: %#v", result)
	}
}

func TestInvokeReservedHostActionIgnoresNormalActions(t *testing.T) {
	_, handled, err := invokeReservedHostAction("provider-a", "completeGrant")
	if err != nil || handled {
		t.Fatalf("normal extension action should pass through: handled=%v err=%v", handled, err)
	}
}
