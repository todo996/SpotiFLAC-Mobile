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
