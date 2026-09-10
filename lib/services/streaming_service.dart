import 'dart:convert';

import 'package:spotiflac_android/services/platform_bridge.dart';

const _hostResolveStreamActionPrefix =
    '__spotiflac_host_resolve_stream_v1__:';
const _streamMediaSourcePrefix = 'spotiflac-stream-v1:';

class StreamMediaRequest {
  final String extensionId;
  final String trackId;
  final String quality;

  const StreamMediaRequest({
    required this.extensionId,
    required this.trackId,
    this.quality = '',
  });

  Map<String, dynamic> toMap() => <String, dynamic>{
    'extension_id': extensionId,
    'track_id': trackId,
    if (quality.isNotEmpty) 'quality': quality,
  };

  static StreamMediaRequest? fromMap(Map<String, dynamic> map) {
    final extensionId = (map['extension_id'] ?? '').toString().trim();
    final trackId = (map['track_id'] ?? '').toString().trim();
    final quality = (map['quality'] ?? '').toString().trim();
    if (extensionId.isEmpty || trackId.isEmpty) return null;
    return StreamMediaRequest(
      extensionId: extensionId,
      trackId: trackId,
      quality: quality,
    );
  }
}

String encodeStreamMediaSource(StreamMediaRequest request) {
  final extensionId = request.extensionId.trim();
  final trackId = request.trackId.trim();
  if (extensionId.isEmpty || trackId.isEmpty) {
    throw const StreamResolutionException(
      'invalid_stream_request',
      'A streaming provider and track identifier are required.',
    );
  }
  final payload = jsonEncode(<String, dynamic>{
    'extension_id': extensionId,
    'track_id': trackId,
    if (request.quality.trim().isNotEmpty) 'quality': request.quality.trim(),
  });
  return '$_streamMediaSourcePrefix${base64Url.encode(utf8.encode(payload)).replaceAll('=', '')}';
}

StreamMediaRequest? decodeStreamMediaSource(String source) {
  if (!source.startsWith(_streamMediaSourcePrefix)) return null;
  final encoded = source.substring(_streamMediaSourcePrefix.length);
  if (encoded.isEmpty) return null;
  try {
    final raw = utf8.decode(base64Url.decode(base64Url.normalize(encoded)));
    final decoded = jsonDecode(raw);
    if (decoded is! Map) return null;
    return StreamMediaRequest.fromMap(Map<String, dynamic>.from(decoded));
  } catch (_) {
    return null;
  }
}

bool isStreamMediaSource(String source) => decodeStreamMediaSource(source) != null;

class ResolvedAudioStream {
  final Uri uri;
  final Map<String, String> headers;
  final String contentType;
  final String provider;
  final String quality;
  final DateTime? expiresAt;

  const ResolvedAudioStream({
    required this.uri,
    this.headers = const {},
    this.contentType = '',
    this.provider = '',
    this.quality = '',
    this.expiresAt,
  });

  factory ResolvedAudioStream.fromMap(Map<String, dynamic> map) {
    if (map['success'] != true) {
      final message = (map['error_message'] ?? map['error'] ?? '').toString();
      final type = (map['error_type'] ?? 'stream_unavailable').toString();
      throw StreamResolutionException(type, message);
    }

    final rawUrl = (map['url'] ?? '').toString().trim();
    final uri = Uri.tryParse(rawUrl);
    if (uri == null ||
        !uri.hasAuthority ||
        (uri.scheme != 'https' && uri.scheme != 'http')) {
      throw const StreamResolutionException(
        'invalid_stream_url',
        'The provider returned an invalid stream URL.',
      );
    }

    final headers = <String, String>{};
    final rawHeaders = map['headers'];
    if (rawHeaders is Map) {
      for (final entry in rawHeaders.entries) {
        final key = entry.key.toString().trim();
        final value = entry.value?.toString() ?? '';
        if (key.isEmpty ||
            key.contains('\r') ||
            key.contains('\n') ||
            value.contains('\r') ||
            value.contains('\n')) {
          throw const StreamResolutionException(
            'invalid_stream_headers',
            'The provider returned invalid stream request headers.',
          );
        }
        headers[key] = value;
      }
    }

    DateTime? expiresAt;
    final expiresAtMs = switch (map['expires_at_ms']) {
      int value => value,
      num value => value.toInt(),
      String value => int.tryParse(value),
      _ => null,
    };
    if (expiresAtMs != null && expiresAtMs > 0) {
      expiresAt = DateTime.fromMillisecondsSinceEpoch(expiresAtMs);
    }

    return ResolvedAudioStream(
      uri: uri,
      headers: Map.unmodifiable(headers),
      contentType: (map['content_type'] ?? '').toString().trim(),
      provider: (map['provider'] ?? '').toString().trim(),
      quality: (map['quality'] ?? '').toString().trim(),
      expiresAt: expiresAt,
    );
  }

  bool get isExpired =>
      expiresAt != null && DateTime.now().isAfter(expiresAt!);

  bool get isExpiringSoon =>
      expiresAt != null &&
      DateTime.now().add(const Duration(seconds: 30)).isAfter(expiresAt!);
}

class StreamResolutionException implements Exception {
  final String type;
  final String message;

  const StreamResolutionException(this.type, this.message);

  @override
  String toString() => message.isEmpty ? type : '$type: $message';
}

class StreamingService {
  const StreamingService._();

  static Future<ResolvedAudioStream> resolveRequest(
    StreamMediaRequest request, {
    Map<String, dynamic>? preparedContext,
  }) => resolve(
    extensionId: request.extensionId,
    trackId: request.trackId,
    quality: request.quality,
    preparedContext: preparedContext,
  );

  static Future<ResolvedAudioStream> resolve({
    required String extensionId,
    required String trackId,
    String quality = '',
    Map<String, dynamic>? preparedContext,
  }) async {
    final normalizedExtensionId = extensionId.trim();
    final normalizedTrackId = trackId.trim();
    if (normalizedExtensionId.isEmpty) {
      throw const StreamResolutionException(
        'missing_provider',
        'A streaming provider is required.',
      );
    }
    if (normalizedTrackId.isEmpty) {
      throw const StreamResolutionException(
        'missing_track',
        'A track identifier is required.',
      );
    }

    final request = <String, dynamic>{
      'track_id': normalizedTrackId,
      if (quality.trim().isNotEmpty) 'quality': quality.trim(),
      if (preparedContext != null && preparedContext.isNotEmpty)
        'prepared_context': preparedContext,
    };
    final payload = base64Url
        .encode(utf8.encode(jsonEncode(request)))
        .replaceAll('=', '');
    final result = await PlatformBridge.invokeExtensionAction(
      normalizedExtensionId,
      '$_hostResolveStreamActionPrefix$payload',
    );
    return ResolvedAudioStream.fromMap(result);
  }
}
