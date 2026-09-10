import 'package:flutter_test/flutter_test.dart';
import 'package:spotiflac_android/services/streaming_service.dart';

void main() {
  group('stream media descriptor', () {
    test('round-trips provider, track and quality without a signed URL', () {
      const request = StreamMediaRequest(
        extensionId: 'provider-a',
        trackId: 'track-123',
        quality: 'LOSSLESS',
      );

      final encoded = encodeStreamMediaSource(request);
      expect(encoded, startsWith('spotiflac-stream-v1:'));
      expect(encoded, isNot(contains('https://')));

      final decoded = decodeStreamMediaSource(encoded);
      expect(decoded, isNotNull);
      expect(decoded!.extensionId, 'provider-a');
      expect(decoded.trackId, 'track-123');
      expect(decoded.quality, 'LOSSLESS');
    });

    test('rejects malformed descriptors', () {
      expect(decodeStreamMediaSource('spotiflac-stream-v1:not-base64'), isNull);
      expect(decodeStreamMediaSource('/music/file.flac'), isNull);
    });
  });

  group('resolved stream validation', () {
    test('accepts https URL, headers and expiry', () {
      final stream = ResolvedAudioStream.fromMap({
        'success': true,
        'url': 'https://cdn.example.com/audio.flac',
        'headers': {'Authorization': 'Bearer token'},
        'content_type': 'audio/flac',
        'provider': 'provider-a',
        'quality': 'LOSSLESS',
        'expires_at_ms': DateTime.now()
            .add(const Duration(minutes: 5))
            .millisecondsSinceEpoch,
      });

      expect(stream.uri.scheme, 'https');
      expect(stream.headers['Authorization'], 'Bearer token');
      expect(stream.contentType, 'audio/flac');
      expect(stream.isExpired, isFalse);
    });

    test('rejects local and unsupported URL schemes', () {
      expect(
        () => ResolvedAudioStream.fromMap({
          'success': true,
          'url': 'file:///tmp/audio.flac',
        }),
        throwsA(isA<StreamResolutionException>()),
      );
    });

    test('rejects CRLF header injection', () {
      expect(
        () => ResolvedAudioStream.fromMap({
          'success': true,
          'url': 'https://cdn.example.com/audio.flac',
          'headers': {'X-Test': 'ok\r\nInjected: yes'},
        }),
        throwsA(isA<StreamResolutionException>()),
      );
    });

    test('reports expired URLs', () {
      final stream = ResolvedAudioStream.fromMap({
        'success': true,
        'url': 'https://cdn.example.com/audio.flac',
        'expires_at_ms': DateTime.now()
            .subtract(const Duration(seconds: 1))
            .millisecondsSinceEpoch,
      });
      expect(stream.isExpired, isTrue);
    });
  });
}
