import 'package:flutter_test/flutter_test.dart';
import 'package:spotiflac_android/models/track.dart';

void main() {
  group('Track.fromBackendMap provider identity', () {
    test('keeps Spotify source id and auxiliary Deezer id for core rows', () {
      final track = Track.fromBackendMap({
        'id': 'fallback-id',
        'spotify_id': 'spotify-123',
        'deezer_id': 'deezer-456',
        'name': 'Example Song',
        'artists': 'Example Artist',
        'album_name': 'Example Album',
        'duration_ms': 201000,
      });

      expect(track.id, 'spotify-123');
      expect(track.source, isNull);
      expect(track.deezerId, 'deezer-456');
    });

    test('keeps extension-native id when a source provider is present', () {
      final track = Track.fromBackendMap({
        'id': 'qobuz-789',
        'spotify_id': 'spotify-123',
        'name': 'Example Song',
        'artists': 'Example Artist',
        'album_name': 'Example Album',
        'duration_ms': 201000,
        'provider_id': 'qobuz',
      });

      expect(track.id, 'qobuz-789');
      expect(track.source, 'qobuz');
    });
  });
}
