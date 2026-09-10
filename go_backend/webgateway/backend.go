package webgateway

import (
	"encoding/json"

	gobackend "github.com/zarz/spotiflac_android/go_backend"
)

// Backend is the small boundary required by the HTTP gateway. Keeping the
// handler behind this interface makes the public HTTP contract testable without
// loading real provider extensions in unit tests.
type Backend interface {
	Search(query string, limit int) (json.RawMessage, error)
	ResolveStream(providerID, trackID, quality string, preparedContext map[string]any) (json.RawMessage, error)
	ProviderCount() int
}

// CoreBackend adapts the existing SpotiFLAC extension runtime to Backend.
type CoreBackend struct{}

func NewCoreBackend() *CoreBackend {
	return &CoreBackend{}
}

func (*CoreBackend) Search(query string, limit int) (json.RawMessage, error) {
	payload, err := gobackend.SearchExtensionTracksJSON(query, limit)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(payload), nil
}

func (*CoreBackend) ResolveStream(providerID, trackID, quality string, preparedContext map[string]any) (json.RawMessage, error) {
	preparedJSON := ""
	if len(preparedContext) > 0 {
		encoded, err := json.Marshal(preparedContext)
		if err != nil {
			return nil, err
		}
		preparedJSON = string(encoded)
	}

	payload, err := gobackend.ResolveExtensionStreamJSON(providerID, trackID, quality, preparedJSON)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(payload), nil
}

func (*CoreBackend) ProviderCount() int {
	return gobackend.GetExtensionMetadataProviderCount()
}
