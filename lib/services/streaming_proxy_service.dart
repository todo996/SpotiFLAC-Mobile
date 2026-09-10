import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'dart:math';

import 'package:spotiflac_android/services/streaming_service.dart';

class StreamingProxyLease {
  final Uri uri;
  final String token;

  const StreamingProxyLease({required this.uri, required this.token});
}

class StreamingProxyService {
  StreamingProxyService._();

  static final StreamingProxyService instance = StreamingProxyService._();

  final HttpClient _client = HttpClient()..autoUncompress = false;
  final Random _random = Random.secure();
  final Map<String, _ProxyEntry> _entries = <String, _ProxyEntry>{};
  HttpServer? _server;
  Future<HttpServer>? _serverStart;

  Future<StreamingProxyLease> open(ResolvedAudioStream stream) async {
    if (stream.isExpired) {
      throw const StreamResolutionException(
        'stream_expired',
        'The resolved stream URL has expired.',
      );
    }

    final server = await _ensureServer();
    _removeExpiredEntries();
    final token = _newToken();
    _entries[token] = _ProxyEntry(stream);
    return StreamingProxyLease(
      uri: Uri.parse('http://127.0.0.1:${server.port}/stream/$token'),
      token: token,
    );
  }

  void release(String token) {
    _entries.remove(token);
  }

  Future<void> close() async {
    _entries.clear();
    final server = _server;
    _server = null;
    _serverStart = null;
    if (server != null) {
      await server.close(force: true);
    }
    _client.close(force: true);
  }

  Future<HttpServer> _ensureServer() async {
    final running = _server;
    if (running != null) return running;
    final starting = _serverStart;
    if (starting != null) return starting;

    final future = HttpServer.bind(
      InternetAddress.loopbackIPv4,
      0,
      shared: false,
    );
    _serverStart = future;
    try {
      final server = await future;
      _server = server;
      unawaited(_serve(server));
      return server;
    } finally {
      _serverStart = null;
    }
  }

  Future<void> _serve(HttpServer server) async {
    await for (final request in server) {
      unawaited(_handle(request));
    }
  }

  Future<void> _handle(HttpRequest request) async {
    try {
      if (request.method != 'GET' && request.method != 'HEAD') {
        request.response.statusCode = HttpStatus.methodNotAllowed;
        await request.response.close();
        return;
      }

      final segments = request.uri.pathSegments;
      if (segments.length != 2 || segments.first != 'stream') {
        request.response.statusCode = HttpStatus.notFound;
        await request.response.close();
        return;
      }

      final token = segments[1];
      final entry = _entries[token];
      if (entry == null) {
        request.response.statusCode = HttpStatus.notFound;
        await request.response.close();
        return;
      }
      if (entry.stream.isExpired) {
        _entries.remove(token);
        request.response.statusCode = HttpStatus.gone;
        await request.response.close();
        return;
      }

      final origin = await _client.openUrl(request.method, entry.stream.uri);
      origin.followRedirects = true;
      origin.maxRedirects = 5;
      origin.headers.set(HttpHeaders.acceptEncodingHeader, 'identity');

      for (final header in entry.stream.headers.entries) {
        origin.headers.set(header.key, header.value);
      }
      for (final name in const <String>[
        HttpHeaders.rangeHeader,
        HttpHeaders.ifRangeHeader,
        HttpHeaders.ifNoneMatchHeader,
        HttpHeaders.ifModifiedSinceHeader,
        HttpHeaders.acceptHeader,
      ]) {
        final values = request.headers[name];
        if (values != null && values.isNotEmpty) {
          origin.headers.set(name, values);
        }
      }

      final upstream = await origin.close();
      final response = request.response;
      response.statusCode = upstream.statusCode;
      upstream.headers.forEach((name, values) {
        if (_isHopByHopHeader(name) ||
            name.toLowerCase() == HttpHeaders.setCookieHeader) {
          return;
        }
        try {
          response.headers.set(name, values);
        } catch (_) {
          // Some platform-managed response headers cannot be set explicitly.
        }
      });

      if (request.method == 'HEAD') {
        await upstream.drain<void>();
        await response.close();
        return;
      }

      await response.addStream(upstream);
      await response.close();
    } catch (_) {
      try {
        request.response.statusCode = HttpStatus.badGateway;
        await request.response.close();
      } catch (_) {}
    }
  }

  bool _isHopByHopHeader(String name) {
    switch (name.toLowerCase()) {
      case 'connection':
      case 'keep-alive':
      case 'proxy-authenticate':
      case 'proxy-authorization':
      case 'te':
      case 'trailer':
      case 'transfer-encoding':
      case 'upgrade':
        return true;
      default:
        return false;
    }
  }

  String _newToken() {
    final bytes = List<int>.generate(24, (_) => _random.nextInt(256));
    return base64Url.encode(bytes).replaceAll('=', '');
  }

  void _removeExpiredEntries() {
    _entries.removeWhere((_, entry) => entry.stream.isExpired);
  }
}

class _ProxyEntry {
  final ResolvedAudioStream stream;

  const _ProxyEntry(this.stream);
}
