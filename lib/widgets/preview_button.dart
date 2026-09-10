import 'package:audio_service/audio_service.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:spotiflac_android/l10n/l10n.dart';
import 'package:spotiflac_android/models/track.dart';
import 'package:spotiflac_android/providers/music_player_provider.dart';
import 'package:spotiflac_android/providers/online_playback_provider.dart';
import 'package:spotiflac_android/providers/preview_player_provider.dart';
import 'package:spotiflac_android/providers/settings_provider.dart';

class PreviewButton extends ConsumerWidget {
  final Track track;
  final double size;

  const PreviewButton({super.key, required this.track, this.size = 24});

  Future<void> _onPressed(BuildContext context, WidgetRef ref) async {
    try {
      await ref.read(previewPlayerProvider.notifier).toggle(track.previewUrl);
    } catch (_) {
      if (!context.mounted) return;
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text(context.l10n.previewUnavailable)));
    }
  }

  /// Loosely matches the built-in player's current item to this track so the
  /// per-track button stays in sync with the mini player instead of showing a
  /// conflicting preview state.
  bool _isCurrentMainTrack(MediaItem? item) {
    if (item == null) return false;
    String norm(String? s) => (s ?? '').toLowerCase().trim();
    return norm(item.title) == norm(track.name) &&
        norm(item.artist) == norm(track.artistName);
  }

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final colorScheme = Theme.of(context).colorScheme;

    // When the built-in player is currently on this track, mirror and control
    // it (consistent with the mini player) rather than the preview snippet.
    final mainItem = ref.watch(currentMediaItemProvider).value;
    if (_isCurrentMainTrack(mainItem)) {
      final isPlaying = ref.watch(playbackPlayingProvider);
      return IconButton(
        iconSize: size,
        padding: EdgeInsets.zero,
        constraints: const BoxConstraints(minWidth: 48, minHeight: 48),
        icon: Icon(
          isPlaying
              ? Icons.pause_circle_filled_rounded
              : Icons.play_circle_fill_rounded,
          color: colorScheme.primary,
        ),
        tooltip: isPlaying
            ? context.l10n.previewStop
            : context.l10n.previewPlay,
        onPressed: () =>
            ref.read(musicPlayerControllerProvider).togglePlayPause(isPlaying),
      );
    }

    if (!track.hasPreview) return const SizedBox.shrink();

    final previewState = ref.watch(previewPlayerProvider);
    final isActive = previewState.isActiveUrl(track.previewUrl);
    final status = isActive ? previewState.status : PreviewStatus.idle;

    final Widget icon;
    final String tooltip;
    switch (status) {
      case PreviewStatus.loading:
        icon = SizedBox(
          width: size * 0.7,
          height: size * 0.7,
          child: CircularProgressIndicator(
            strokeWidth: 2,
            color: colorScheme.primary,
          ),
        );
        tooltip = context.l10n.previewStop;
        break;
      case PreviewStatus.playing:
        icon = Icon(
          Icons.pause_circle_filled_rounded,
          color: colorScheme.primary,
        );
        tooltip = context.l10n.previewStop;
        break;
      case PreviewStatus.paused:
        icon = Icon(Icons.play_circle_fill_rounded, color: colorScheme.primary);
        tooltip = context.l10n.previewPlay;
        break;
      case PreviewStatus.idle:
        icon = Icon(
          Icons.play_circle_outline_rounded,
          color: colorScheme.onSurfaceVariant,
        );
        tooltip = context.l10n.previewPlay;
        break;
    }

    return IconButton(
      iconSize: size,
      padding: EdgeInsets.zero,
      constraints: const BoxConstraints(minWidth: 48, minHeight: 48),
      icon: icon,
      tooltip: tooltip,
      onPressed: () => _onPressed(context, ref),
    );
  }
}

/// Full-track online playback action. This is intentionally separate from
/// [PreviewButton] and from the row's existing download tap behavior so adding
/// streaming never changes how existing downloads work.
class OnlinePlayButton extends ConsumerStatefulWidget {
  final Track track;
  final double size;

  const OnlinePlayButton({super.key, required this.track, this.size = 24});

  @override
  ConsumerState<OnlinePlayButton> createState() => _OnlinePlayButtonState();
}

class _OnlinePlayButtonState extends ConsumerState<OnlinePlayButton> {
  bool _starting = false;

  bool _isCurrentTrack(MediaItem? item) {
    if (item == null) return false;
    String norm(String? value) => (value ?? '').toLowerCase().trim();
    return norm(item.title) == norm(widget.track.name) &&
        norm(item.artist) == norm(widget.track.artistName);
  }

  Future<void> _play() async {
    if (_starting) return;
    setState(() => _starting = true);
    try {
      await ref.read(onlinePlaybackProvider.notifier).playTrack(widget.track);
    } catch (error) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(context.friendlyError(error))),
      );
    } finally {
      if (mounted) setState(() => _starting = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final provider = ref.watch(
      settingsProvider.select((settings) => settings.defaultService.trim()),
    );
    final fallbackProvider = (widget.track.source ?? '').trim();
    if (provider.isEmpty && fallbackProvider.isEmpty) {
      return const SizedBox.shrink();
    }

    final colorScheme = Theme.of(context).colorScheme;
    final currentItem = ref.watch(currentMediaItemProvider).value;
    final isCurrent = _isCurrentTrack(currentItem);
    final isPlaying = isCurrent && ref.watch(playbackPlayingProvider);
    final isLoading = isCurrent && ref.watch(playbackLoadingProvider);

    if (_starting || isLoading) {
      return SizedBox(
        width: 48,
        height: 48,
        child: Center(
          child: SizedBox(
            width: widget.size * 0.7,
            height: widget.size * 0.7,
            child: CircularProgressIndicator(
              strokeWidth: 2,
              color: colorScheme.primary,
            ),
          ),
        ),
      );
    }

    return IconButton(
      iconSize: widget.size,
      padding: EdgeInsets.zero,
      constraints: const BoxConstraints(minWidth: 48, minHeight: 48),
      icon: Icon(
        isPlaying
            ? Icons.pause_circle_filled_rounded
            : Icons.play_circle_fill_rounded,
        color: colorScheme.primary,
      ),
      // Reuse the existing localized transport labels until dedicated online
      // playback strings are added to every locale.
      tooltip: isPlaying ? context.l10n.previewStop : context.l10n.previewPlay,
      onPressed: isCurrent
          ? () => ref
                .read(musicPlayerControllerProvider)
                .togglePlayPause(isPlaying)
          : _play,
    );
  }
}
