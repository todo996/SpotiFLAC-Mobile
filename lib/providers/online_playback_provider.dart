import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:spotiflac_android/models/track.dart';
import 'package:spotiflac_android/providers/music_player_provider.dart';
import 'package:spotiflac_android/providers/settings_provider.dart';
import 'package:spotiflac_android/services/music_player_service.dart';
import 'package:spotiflac_android/services/streaming_service.dart';

class OnlinePlaybackState {
  const OnlinePlaybackState();
}

class OnlinePlaybackController extends Notifier<OnlinePlaybackState> {
  @override
  OnlinePlaybackState build() => const OnlinePlaybackState();

  String _providerFor(Track track, {String? providerId}) {
    final explicit = providerId?.trim() ?? '';
    if (explicit.isNotEmpty) return explicit;

    // Keep provider selection consistent with the existing download pipeline.
    // defaultService is reconciled by ExtensionNotifier so stale/disabled
    // download extensions are replaced or cleared there.
    final configured = ref.read(settingsProvider).defaultService.trim();
    if (configured.isNotEmpty) return configured;

    // Extension-native search results may already carry a provider id. This is
    // a final additive fallback only; the resolver will still validate that it
    // is an enabled download provider with resolveStream support.
    return (track.source ?? '').trim();
  }

  PlayableMedia _toPlayable(
    Track track, {
    String? providerId,
    String? quality,
  }) {
    final provider = _providerFor(track, providerId: providerId);
    if (provider.isEmpty) {
      throw const StreamResolutionException(
        'missing_provider',
        'No download provider is available for online playback.',
      );
    }

    final requestedQuality = (quality ?? ref.read(settingsProvider).audioQuality)
        .trim();
    return PlayableMedia(
      id: 'stream:$provider:${track.id}',
      source: encodeStreamMediaSource(
        StreamMediaRequest(
          extensionId: provider,
          trackId: track.id,
          quality: requestedQuality,
        ),
      ),
      title: track.name,
      artist: track.artistName,
      album: track.albumName,
      artUri: (track.coverUrl ?? '').trim().isEmpty ? null : track.coverUrl,
      duration: track.duration > 0 ? Duration(seconds: track.duration) : null,
      format: track.audioQuality,
      explicit: track.isExplicit,
    );
  }

  Future<void> playTrack(
    Track track, {
    String? providerId,
    String? quality,
  }) async {
    final handler = await ref
        .read(musicPlayerControllerProvider)
        .ensureInitialized();
    if (handler == null) {
      throw const StreamResolutionException(
        'player_unavailable',
        'The built-in player could not be initialized.',
      );
    }
    await handler.setQueueAndPlay([
      _toPlayable(track, providerId: providerId, quality: quality),
    ]);
  }

  Future<void> playTrackList(
    List<Track> tracks, {
    int startIndex = 0,
    String? providerId,
    String? quality,
  }) async {
    if (tracks.isEmpty) return;
    final items = tracks
        .map(
          (track) => _toPlayable(
            track,
            providerId: providerId,
            quality: quality,
          ),
        )
        .toList(growable: false);
    final handler = await ref
        .read(musicPlayerControllerProvider)
        .ensureInitialized();
    if (handler == null) {
      throw const StreamResolutionException(
        'player_unavailable',
        'The built-in player could not be initialized.',
      );
    }
    await handler.setQueueAndPlay(
      items,
      initialIndex: startIndex.clamp(0, items.length - 1),
    );
  }

  Future<void> playNext(
    Track track, {
    String? providerId,
    String? quality,
  }) async {
    final handler = await ref
        .read(musicPlayerControllerProvider)
        .ensureInitialized();
    if (handler == null) return;
    await handler.enqueue(
      _toPlayable(track, providerId: providerId, quality: quality),
      playNext: true,
    );
  }

  Future<void> addToQueue(
    Track track, {
    String? providerId,
    String? quality,
  }) async {
    final handler = await ref
        .read(musicPlayerControllerProvider)
        .ensureInitialized();
    if (handler == null) return;
    await handler.enqueue(
      _toPlayable(track, providerId: providerId, quality: quality),
    );
  }
}

final onlinePlaybackProvider =
    NotifierProvider<OnlinePlaybackController, OnlinePlaybackState>(
      OnlinePlaybackController.new,
    );
