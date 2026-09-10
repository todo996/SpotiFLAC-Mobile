from pathlib import Path
import sys

root = Path(sys.argv[1] if len(sys.argv) > 1 else ".")


def replace_once(relative_path: str, old: str, new: str, label: str) -> None:
    path = root / relative_path
    text = path.read_text()
    if new in text:
        print(f"{label}: already applied")
        return
    if old not in text:
        raise SystemExit(f"{label}: anchor not found; refusing unsafe patch")
    path.write_text(text.replace(old, new, 1))
    print(f"{label}: applied")


player = "lib/services/music_player_service.dart"
replace_once(
    player,
    "import 'package:spotiflac_android/services/playback_normalization.dart';\nimport 'package:spotiflac_android/utils/int_utils.dart';",
    "import 'package:spotiflac_android/services/playback_normalization.dart';\nimport 'package:spotiflac_android/services/streaming_proxy_service.dart';\nimport 'package:spotiflac_android/services/streaming_service.dart';\nimport 'package:spotiflac_android/utils/int_utils.dart';",
    "player imports",
)
replace_once(
    player,
    "  bool get isContentUri => source.startsWith('content://');\n",
    "  bool get isContentUri => source.startsWith('content://');\n\n  bool get isStreamSource => isStreamMediaSource(source);\n",
    "stream source getter",
)
replace_once(
    player,
    "  String? _activeResolvedPath;\n  bool _disposed = false;",
    "  String? _activeResolvedPath;\n  String? _activeStreamingProxyToken;\n  bool _disposed = false;",
    "stream proxy state",
)
replace_once(
    player,
    "      final media = _media[index];\n      var resolved = media.isContentUri",
    "      final media = _media[index];\n      if (media.isStreamSource) {\n        if (_index == index &&\n            generation == _playRequestGeneration &&\n            normalizationGeneration == _normalizationGeneration) {\n          try {\n            await _player.setVolume(1.0);\n          } catch (e) {\n            _log.w('Failed to reset streaming volume: $e');\n          }\n        }\n        return;\n      }\n      var resolved = media.isContentUri",
    "skip local ReplayGain for streams",
)
replace_once(
    player,
    "  Future<String?> _resolveSource(PlayableMedia media) async {\n    if (!media.isContentUri) return media.source;",
    "  Future<String?> _resolveSource(PlayableMedia media) async {\n    if (media.isStreamSource) return null;\n    if (!media.isContentUri) return media.source;",
    "keep stream descriptors out of local resolver",
)
replace_once(
    player,
    "  Future<void> _cleanupPendingResolvedPaths() async {\n    final deletable = _pendingResolvedPathDeletes",
    "  void _releaseStreamingProxy() {\n    final token = _activeStreamingProxyToken;\n    _activeStreamingProxyToken = null;\n    if (token != null) {\n      StreamingProxyService.instance.release(token);\n    }\n  }\n\n  Future<({Source source, String? proxyToken})> _sourceForResolvedStream(\n    ResolvedAudioStream stream,\n  ) async {\n    final mimeType = stream.contentType.isEmpty ? null : stream.contentType;\n    if (stream.headers.isEmpty && stream.uri.scheme == 'https') {\n      return (\n        source: UrlSource(stream.uri.toString(), mimeType: mimeType),\n        proxyToken: null,\n      );\n    }\n    final lease = await StreamingProxyService.instance.open(stream);\n    return (\n      source: UrlSource(lease.uri.toString(), mimeType: mimeType),\n      proxyToken: lease.token,\n    );\n  }\n\n  Future<void> _cleanupPendingResolvedPaths() async {\n    final deletable = _pendingResolvedPathDeletes",
    "stream source/proxy helpers",
)
# Patch the existing local playback path before adding the streaming method so
# this anchor cannot accidentally match the new online branch.
replace_once(
    player,
    "      await _activateAudioSession();\n      if (!_isCurrentPlayRequest(generation, media)) return;\n      await _player.stop();\n      _sourceReady = false;",
    "      await _activateAudioSession();\n      if (!_isCurrentPlayRequest(generation, media)) return;\n      await _player.stop();\n      _releaseStreamingProxy();\n      _sourceReady = false;",
    "release old stream before local playback",
)
stream_method = r'''  Future<void> _playStreamingMedia(
    int index,
    int generation,
    PlayableMedia media,
    Duration effectiveStartPosition,
  ) async {
    final request = decodeStreamMediaSource(media.source);
    if (request == null) {
      _log.e('Invalid stream descriptor for ${media.title}');
      _broadcastState(playerState: PlayerState.stopped);
      return;
    }

    try {
      await musicPlayerExclusiveAudioHook?.call();
    } catch (_) {}
    if (!_isCurrentPlayRequest(generation, media)) return;

    _switchingGeneration = generation;
    String? pendingProxyToken;
    try {
      var stream = await StreamingService.resolveRequest(request);
      if (stream.isExpiringSoon) {
        stream = await StreamingService.resolveRequest(request);
      }
      if (!_isCurrentPlayRequest(generation, media)) return;

      await _player.setAudioContext(_musicAudioContext);
      if (!_isCurrentPlayRequest(generation, media)) return;
      await _activateAudioSession();
      if (!_isCurrentPlayRequest(generation, media)) return;
      await _player.stop();
      _releaseStreamingProxy();
      _activeResolvedPath = null;
      _sourceReady = false;
      await _player.setVolume(1.0);
      if (!_isCurrentPlayRequest(generation, media)) return;

      final startAt = effectiveStartPosition > Duration.zero
          ? effectiveStartPosition
          : null;
      var prepared = await _sourceForResolvedStream(stream);
      pendingProxyToken = prepared.proxyToken;
      try {
        await _player.play(prepared.source, position: startAt);
      } catch (firstError) {
        if (pendingProxyToken != null) {
          StreamingProxyService.instance.release(pendingProxyToken!);
          pendingProxyToken = null;
        }
        if (!_isCurrentPlayRequest(generation, media)) rethrow;
        _log.w(
          'Streaming source failed; resolving a fresh URL once: $firstError',
        );
        await _player.stop();
        final retryStream = await StreamingService.resolveRequest(request);
        if (!_isCurrentPlayRequest(generation, media)) return;
        prepared = await _sourceForResolvedStream(retryStream);
        pendingProxyToken = prepared.proxyToken;
        await _player.play(prepared.source, position: startAt);
      }

      if (!_isCurrentPlayRequest(generation, media)) return;
      _activeStreamingProxyToken = pendingProxyToken;
      pendingProxyToken = null;
      _sourceReady = true;
      _pendingRestorePosition = null;
      _broadcastPosition(effectiveStartPosition, force: true);
      _broadcastState(playerState: PlayerState.playing);
      _lastPeriodicPersistAt = DateTime.now();
      unawaited(_persistSession(position: effectiveStartPosition));
      unawaited(_ensureDurationKnown(index, generation));
    } catch (e) {
      if (!_isCurrentPlayRequest(generation, media)) return;
      _sourceReady = false;
      _log.e('Streaming playback failed for ${media.title}: $e');
      _broadcastState(playerState: PlayerState.stopped);
    } finally {
      if (pendingProxyToken != null) {
        StreamingProxyService.instance.release(pendingProxyToken!);
      }
      if (_switchingGeneration == generation) {
        _switchingGeneration = 0;
      }
    }
  }

'''
replace_once(
    player,
    "  Future<void> _playIndex(\n    int index, {",
    stream_method + "  Future<void> _playIndex(\n    int index, {",
    "stream play engine",
)
replace_once(
    player,
    "    await _claimHardwareMediaButtons();\n    if (!_isCurrentPlayRequest(generation, media)) return;\n\n    // Android opens SAF files",
    "    await _claimHardwareMediaButtons();\n    if (!_isCurrentPlayRequest(generation, media)) return;\n\n    if (media.isStreamSource) {\n      await _playStreamingMedia(\n        index,\n        generation,\n        media,\n        effectiveStartPosition,\n      );\n      return;\n    }\n\n    // Android opens SAF files",
    "route stream media to online engine",
)
replace_once(
    player,
    "    await _player.stop();\n    _sourceReady = false;\n    _activeResolvedPath = null;\n    await _cleanupPendingResolvedPaths();",
    "    await _player.stop();\n    _releaseStreamingProxy();\n    _sourceReady = false;\n    _activeResolvedPath = null;\n    await _cleanupPendingResolvedPaths();",
    "release proxy on stop",
)
replace_once(
    player,
    "    await _player.dispose();\n    _activeResolvedPath = null;",
    "    await _player.dispose();\n    _releaseStreamingProxy();\n    _activeResolvedPath = null;",
    "release proxy on dispose",
)
replace_once(
    player,
    "      if (!media.isContentUri && !await File(media.source).exists()) {\n        continue;\n      }",
    "      if (!media.isContentUri &&\n          !media.isStreamSource &&\n          !await File(media.source).exists()) {\n        continue;\n      }",
    "restore stable stream descriptors",
)

now_playing = "lib/screens/now_playing_screen.dart"
replace_once(
    now_playing,
    "import 'package:spotiflac_android/services/music_player_service.dart';\nimport 'package:spotiflac_android/utils/clickable_metadata.dart';",
    "import 'package:spotiflac_android/services/music_player_service.dart';\nimport 'package:spotiflac_android/services/streaming_service.dart';\nimport 'package:spotiflac_android/utils/clickable_metadata.dart';",
    "now playing stream import",
)
replace_once(
    now_playing,
    "    final sameItem =\n        source == _loadedSource &&\n        effectiveResolvedSource == _loadedResolvedSource;\n\n    if (sameItem) {",
    "    final sameItem =\n        source == _loadedSource &&\n        effectiveResolvedSource == _loadedResolvedSource;\n\n    if (isStreamMediaSource(source)) {\n      _loadedSource = source;\n      _loadedResolvedSource = effectiveResolvedSource;\n      _loadedMetadataPath = null;\n      if (mounted) {\n        setState(() {\n          _loadingMeta = false;\n          _metadata = fallbackMetadata.isEmpty ? null : fallbackMetadata;\n          _lyrics = ParsedLyrics.empty;\n        });\n      }\n      return;\n    }\n\n    if (sameItem) {",
    "now playing stream metadata guard",
)

home_widgets = "lib/screens/home_tab_widgets.dart"
replace_once(
    home_widgets,
    "                PreviewButton(track: track),\n                TrackCollectionQuickActions(",
    "                OnlinePlayButton(track: track),\n                PreviewButton(track: track),\n                TrackCollectionQuickActions(",
    "search result online play button",
)

album = "lib/screens/album_screen.dart"
replace_once(
    album,
    "import 'package:spotiflac_android/providers/extension_provider.dart';\nimport 'package:spotiflac_android/providers/recent_access_provider.dart';",
    "import 'package:spotiflac_android/providers/extension_provider.dart';\nimport 'package:spotiflac_android/providers/online_playback_provider.dart';\nimport 'package:spotiflac_android/providers/recent_access_provider.dart';",
    "album online player import",
)
replace_once(
    album,
    "              children: [\n                _buildLoveAllButton(),\n                const SizedBox(width: 12),\n                Flexible(\n                  child: HeaderFilledButton(",
    "              children: [\n                HeaderCircleButton(\n                  icon: Icons.play_arrow_rounded,\n                  tooltip: context.l10n.previewPlay,\n                  onPressed: tracks.isEmpty ? null : () => _playAll(tracks),\n                ),\n                const SizedBox(width: 12),\n                _buildLoveAllButton(),\n                const SizedBox(width: 12),\n                Flexible(\n                  child: HeaderFilledButton(",
    "album play all action",
)
replace_once(
    album,
    "  Widget _buildLoveAllButton() {",
    "  Future<void> _playAll(List<Track> tracks) async {\n    try {\n      await ref.read(onlinePlaybackProvider.notifier).playTrackList(\n        tracks,\n        providerId: _recommendedDownloadService(),\n      );\n    } catch (error) {\n      if (!mounted) return;\n      ScaffoldMessenger.of(context).showSnackBar(\n        SnackBar(content: Text(context.friendlyError(error))),\n      );\n    }\n  }\n\n  Widget _buildLoveAllButton() {",
    "album play all handler",
)

playlist = "lib/screens/playlist_screen.dart"
replace_once(
    playlist,
    "import 'package:spotiflac_android/providers/library_collections_provider.dart';\nimport 'package:spotiflac_android/utils/image_cache_utils.dart';",
    "import 'package:spotiflac_android/providers/library_collections_provider.dart';\nimport 'package:spotiflac_android/providers/online_playback_provider.dart';\nimport 'package:spotiflac_android/utils/image_cache_utils.dart';",
    "playlist online player import",
)
replace_once(
    playlist,
    "        children: [\n          _buildLoveAllButton(),\n          const SizedBox(width: 12),\n          Flexible(child: _buildDownloadAllCenterButton(context)),",
    "        children: [\n          HeaderCircleButton(\n            icon: Icons.play_arrow_rounded,\n            tooltip: context.l10n.previewPlay,\n            onPressed: _tracks.isEmpty ? null : _playAll,\n          ),\n          const SizedBox(width: 12),\n          _buildLoveAllButton(),\n          const SizedBox(width: 12),\n          Flexible(child: _buildDownloadAllCenterButton(context)),",
    "playlist play all action",
)
replace_once(
    playlist,
    "  Widget _buildLoveAllButton() {",
    "  Future<void> _playAll() async {\n    try {\n      await ref.read(onlinePlaybackProvider.notifier).playTrackList(\n        _tracks,\n        providerId: _recommendedDownloadService(),\n      );\n    } catch (error) {\n      if (!mounted) return;\n      ScaffoldMessenger.of(context).showSnackBar(\n        SnackBar(content: Text(context.friendlyError(error))),\n      );\n    }\n  }\n\n  Widget _buildLoveAllButton() {",
    "playlist play all handler",
)

print("Mobile streaming integration patch completed")
