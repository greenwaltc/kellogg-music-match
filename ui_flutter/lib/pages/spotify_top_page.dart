import 'package:flutter/material.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:url_launcher/url_launcher.dart';

import '../services/api_client.dart';
import '../services/spotify_top_service.dart';
import 'spotify_connect_prompt.dart';

class SpotifyTopPage extends StatefulWidget {
  const SpotifyTopPage({super.key, this.onNeedConnect});

  final VoidCallback? onNeedConnect;

  @override
  State<SpotifyTopPage> createState() => _SpotifyTopPageState();
}

class _SpotifyTopPageState extends State<SpotifyTopPage> {
  late final ApiClient _api;
  SharedPreferences? _prefs;
  SpotifyTopService? _svc;
  final ScrollController _scrollController = ScrollController();

  String _basis = 'artists'; // 'artists' | 'tracks'
  String _range = 'medium_term'; // short_term | medium_term | long_term
  final int _pageSize = 20;

  bool _loading = false;
  bool _loadingMore = false;
  String? _error;
  int _offset = 0;
  int _totalCount = 0;
  bool _hasMore = true;
  final List<Map<String, dynamic>> _items = [];
  bool _syncing = false;
  int _syncProgress = 0; // 0..100
  int _syncCountdown = 0; // seconds remaining
  DateTime? _lastSyncAt;
  bool _canCancel = false;
  bool _mounted = true;
  bool _showScrollToTop = false;

  @override
  void initState() {
    super.initState();
    _scrollController.addListener(_onScroll);
    _init();
  }

  void _onScroll() {
    final show = _scrollController.positions.isNotEmpty
        ? _scrollController.offset > 300
        : false;
    if (show != _showScrollToTop) {
      setState(() => _showScrollToTop = show);
    }
  }

  Future<void> _init() async {
    final prefs = await SharedPreferences.getInstance();
    setState(() {
      _api = ApiClient();
      _prefs = prefs;
      _svc = SpotifyTopService(_api, prefs);
      _basis = prefs.getString('spotify_basis') ?? 'artists';
      _range = prefs.getString('spotify_range') ?? 'medium_term';
    });
    // Load initial sync status to show last finished time
    _loadInitialStatus();
    _refresh();
  }

  Future<void> _loadInitialStatus() async {
    try {
      final st = await _svc?.getSpotifySyncStatus();
      if (st != null) {
        final finishedAt = st['finishedAt'] as String?;
        if (finishedAt != null && finishedAt.isNotEmpty) {
          setState(
            () => _lastSyncAt = DateTime.tryParse(finishedAt)?.toLocal(),
          );
        }
      }
    } catch (_) {}
  }

  Future<void> _refresh() async {
    if (_svc == null) return;
    setState(() {
      _loading = true;
      _error = null;
      _items.clear();
      _offset = 0;
      _totalCount = 0;
      _hasMore = true;
    });
    try {
      final page = await _fetch();
      setState(() {
        _items.addAll((page['items'] as List).cast<Map<String, dynamic>>());
        _hasMore = page['hasMore'] as bool? ?? false;
        _totalCount = (page['totalCount'] as num?)?.toInt() ?? _items.length;
        _offset = _items.length;
      });
    } catch (e) {
      setState(() => _error = e.toString());
    } finally {
      setState(() => _loading = false);
    }
  }

  Future<Map<String, dynamic>> _fetch() async {
    if (_svc == null) throw StateError('Service not ready');
    if (_basis == 'tracks') {
      return await _svc!.fetchTopTracksPage(
        range: _range,
        limit: _pageSize,
        offset: _offset,
      );
    } else {
      return await _svc!.fetchTopArtistsPage(
        range: _range,
        limit: _pageSize,
        offset: _offset,
      );
    }
  }

  Future<void> _loadMore() async {
    if (_loadingMore || !_hasMore) return;
    setState(() => _loadingMore = true);
    try {
      final page = await _fetch();
      setState(() {
        _items.addAll((page['items'] as List).cast<Map<String, dynamic>>());
        _hasMore = page['hasMore'] as bool? ?? false;
        _totalCount = (page['totalCount'] as num?)?.toInt() ?? _items.length;
        _offset = _items.length;
      });
    } catch (e) {
      setState(() => _error = e.toString());
    } finally {
      setState(() => _loadingMore = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final topControls = _buildControls(context);
    final list = _buildList(context);
    return Column(
      children: [
        Material(
          elevation: 2,
          color: Theme.of(context).colorScheme.surface,
          child: SafeArea(bottom: false, child: topControls),
        ),
        Expanded(
          child: Stack(
            children: [
              list,
              Positioned(
                right: 16,
                bottom: 16,
                child: AnimatedScale(
                  duration: const Duration(milliseconds: 150),
                  scale: _showScrollToTop ? 1.0 : 0.0,
                  child: AnimatedOpacity(
                    duration: const Duration(milliseconds: 150),
                    opacity: _showScrollToTop ? 1.0 : 0.0,
                    child: FloatingActionButton(
                      mini: true,
                      onPressed: () {
                        if (_scrollController.hasClients) {
                          _scrollController.animateTo(
                            0,
                            duration: const Duration(milliseconds: 300),
                            curve: Curves.easeOutCubic,
                          );
                        }
                      },
                      tooltip: 'Scroll to top',
                      child: const Icon(Icons.arrow_upward),
                    ),
                  ),
                ),
              ),
            ],
          ),
        ),
      ],
    );
  }

  Widget _buildControls(BuildContext context) {
    final basisToggle = SegmentedButton<String>(
      segments: const <ButtonSegment<String>>[
        ButtonSegment<String>(
          value: 'artists',
          label: Text('Artists'),
          icon: Icon(Icons.person),
        ),
        ButtonSegment<String>(
          value: 'tracks',
          label: Text('Tracks'),
          icon: Icon(Icons.music_note),
        ),
      ],
      selected: {_basis},
      onSelectionChanged: (s) {
        setState(() => _basis = s.first);
        _prefs?.setString('spotify_basis', _basis);
        _refresh();
      },
    );
    final rangeToggle = SegmentedButton<String>(
      segments: const <ButtonSegment<String>>[
        ButtonSegment<String>(value: 'short_term', label: Text('Last 4 weeks')),
        ButtonSegment<String>(
          value: 'medium_term',
          label: Text('Last 6 months'),
        ),
        ButtonSegment<String>(
          value: 'long_term',
          label: Text('Last few years'),
        ),
      ],
      selected: {_range},
      onSelectionChanged: (s) {
        setState(() => _range = s.first);
        _prefs?.setString('spotify_range', _range);
        _refresh();
      },
    );

    final countText = _totalCount > 0
        ? Text(
            '$_totalCount results',
            textAlign: TextAlign.right,
            style: TextStyle(color: Theme.of(context).hintColor),
          )
        : const SizedBox.shrink();

    final refreshButton = OutlinedButton.icon(
      onPressed: _syncing ? null : _onRefreshFromSpotify,
      icon: _syncing
          ? SizedBox(
              width: 16,
              height: 16,
              child: CircularProgressIndicator(strokeWidth: 2),
            )
          : const Icon(Icons.refresh),
      label: const Text('Refresh from Spotify'),
    );

    final cancelButton = TextButton.icon(
      onPressed: _syncing && _canCancel ? _onCancelSync : null,
      icon: const Icon(Icons.close),
      label: const Text('Cancel'),
    );

    final lastSyncText = _lastSyncAt != null
        ? Text(
            'Last sync: ${_formatTime(_lastSyncAt!)}',
            style: TextStyle(color: Theme.of(context).hintColor),
          )
        : const SizedBox.shrink();

    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 12, 16, 12),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          basisToggle,
          const SizedBox(height: 8),
          rangeToggle,
          Row(
            children: [
              Expanded(
                child: Wrap(
                  spacing: 8,
                  runSpacing: 8,
                  crossAxisAlignment: WrapCrossAlignment.center,
                  children: [
                    refreshButton,
                    if (_syncing && _canCancel) cancelButton,
                  ],
                ),
              ),
              const SizedBox(width: 12),
              Column(
                crossAxisAlignment: CrossAxisAlignment.end,
                children: [lastSyncText, countText],
              ),
            ],
          ),
          if (_syncing) ...[
            const SizedBox(height: 8),
            Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                LinearProgressIndicator(
                  value: _syncProgress > 0 ? _syncProgress / 100.0 : null,
                ),
                const SizedBox(height: 4),
                Text(
                  _syncCountdown > 0
                      ? 'Syncing… $_syncProgress% · ${_syncCountdown}s remaining'
                      : 'Syncing… $_syncProgress%',
                  textAlign: TextAlign.right,
                  style: TextStyle(color: Theme.of(context).hintColor),
                ),
              ],
            ),
          ],
          if (_error != null) ...[
            const SizedBox(height: 8),
            Text(
              _error!,
              style: TextStyle(color: Theme.of(context).colorScheme.error),
            ),
          ],
        ],
      ),
    );
  }

  Future<void> _onRefreshFromSpotify() async {
    if (_svc == null) return;
    setState(() {
      _syncing = true;
      _syncProgress = 0;
      _syncCountdown = 30; // seconds
      _canCancel = true;
    });
    if (mounted) {
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(const SnackBar(content: Text('Starting Spotify sync…')));
    }
    try {
      await _svc!.refreshFromSpotify();
      await _pollSyncStatus(timeoutSeconds: 30);
      await _refresh();
    } catch (e) {
      // If no stored tokens, prompt user to reconnect Spotify instead of generic failure
      if (e is ApiException && e.status == 404) {
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            const SnackBar(
              content: Text(
                'No stored Spotify tokens found. Please connect Spotify first.',
              ),
            ),
          );
        }
        widget.onNeedConnect?.call();
      } else {
        if (mounted) {
          setState(() => _error = e.toString());
          ScaffoldMessenger.of(
            context,
          ).showSnackBar(SnackBar(content: Text('Spotify sync failed: $e')));
        }
      }
    } finally {
      if (mounted) setState(() => _syncing = false);
    }
  }

  Future<void> _onCancelSync() async {
    if (_svc == null) return;
    try {
      await _svc!.cancelSpotifySync();
      setState(() {
        _canCancel = false;
      });
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(const SnackBar(content: Text('Sync cancelled')));
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text('Failed to cancel: $e')));
      }
    }
  }

  Future<void> _pollSyncStatus({required int timeoutSeconds}) async {
    final start = DateTime.now();
    while (true) {
      await Future.delayed(const Duration(milliseconds: 500));
      if (!_mounted) break;
      try {
        final st = await _svc!.getSpotifySyncStatus();
        final status = (st['status'] as String?) ?? '';
        final prog = (st['progress'] as num?)?.toInt() ?? _syncProgress;
        final finishedAt = st['finishedAt'] as String?;
        setState(() {
          _syncProgress = prog.clamp(0, 100);
          final elapsed = DateTime.now().difference(start).inSeconds;
          _syncCountdown = (timeoutSeconds - elapsed).clamp(0, timeoutSeconds);
        });
        if (finishedAt != null && finishedAt.isNotEmpty) {
          setState(
            () => _lastSyncAt = DateTime.tryParse(finishedAt)?.toLocal(),
          );
        }
        if (status == 'complete' ||
            status == 'failed' ||
            status == 'cancelled') {
          return;
        }
      } catch (_) {
        // Ignore transient errors
      }
      if (DateTime.now().difference(start).inSeconds >= timeoutSeconds) {
        return;
      }
    }
  }

  String _formatTime(DateTime dt) {
    final now = DateTime.now();
    final diff = now.difference(dt);
    if (diff.inMinutes < 1) return 'just now';
    if (diff.inMinutes < 60) return '${diff.inMinutes}m ago';
    if (diff.inHours < 24) return '${diff.inHours}h ago';
    return '${dt.month}/${dt.day}/${dt.year} ${dt.hour.toString().padLeft(2, '0')}:${dt.minute.toString().padLeft(2, '0')}';
  }

  @override
  void dispose() {
    _mounted = false;
    _scrollController.removeListener(_onScroll);
    _scrollController.dispose();
    super.dispose();
  }

  Widget _buildList(BuildContext context) {
    if (_loading && _items.isEmpty) {
      return const Center(child: CircularProgressIndicator());
    }
    if (_error != null && _items.isEmpty) {
      // If unauthorized due to missing connection, show connect prompt inline
      return ListView(
        padding: const EdgeInsets.symmetric(vertical: 24),
        children: [
          Center(
            child: Column(
              children: const [
                Icon(Icons.link_off, size: 48),
                SizedBox(height: 8),
                Text('Unable to load Spotify data.'),
              ],
            ),
          ),
          SpotifyConnectPrompt(onConnected: _refresh),
        ],
      );
    }
    return NotificationListener<ScrollNotification>(
      onNotification: (n) {
        // Toggle scroll-to-top button based on vertical offset
        final current = n.metrics.pixels > 200;
        if (current != _showScrollToTop) {
          setState(() => _showScrollToTop = current);
        }
        if (n.metrics.pixels >= n.metrics.maxScrollExtent - 200) {
          _loadMore();
        }
        return false;
      },
      child: RefreshIndicator(
        onRefresh: _refresh,
        child: ListView.builder(
          controller: _scrollController,
          padding: const EdgeInsets.only(top: 8, bottom: 16),
          itemCount: _items.length + (_hasMore ? 1 : 0),
          itemBuilder: (context, index) {
            if (index >= _items.length) {
              return const Padding(
                padding: EdgeInsets.all(16),
                child: Center(child: CircularProgressIndicator()),
              );
            }
            final it = _items[index];
            return _SpotifyRow(basis: _basis, item: it);
          },
        ),
      ),
    );
  }
}

class _SpotifyRow extends StatelessWidget {
  const _SpotifyRow({required this.basis, required this.item});

  final String basis;
  final Map<String, dynamic> item;

  @override
  Widget build(BuildContext context) {
    final rank = (item['rank'] as num?)?.toInt() ?? 0;
    final name = (item['name'] as String?) ?? '';
    final imageUrl =
        (item['imageUrl'] ?? item['imageURL'] ?? item['image_url']) as String?;
    final subtitle = basis == 'tracks' && item['artistNames'] is List
        ? (item['artistNames'] as List).cast<String>().join(', ')
        : null;

    final rankBox = SizedBox(
      width: 36,
      child: Center(
        child: Text(
          rank > 0 ? '$rank' : '•',
          style: const TextStyle(fontWeight: FontWeight.w600),
        ),
      ),
    );

    final image = ClipRRect(
      borderRadius: BorderRadius.circular(6),
      child: imageUrl != null && imageUrl.isNotEmpty
          ? Image.network(
              imageUrl,
              width: 44,
              height: 44,
              fit: BoxFit.cover,
              errorBuilder: (_, __, ___) => _placeholderBox(context),
            )
          : _placeholderBox(context, size: 44),
    );

    final title = Expanded(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            name,
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
            style: const TextStyle(fontSize: 16, fontWeight: FontWeight.w500),
          ),
          if (subtitle != null && subtitle.isNotEmpty)
            Text(
              subtitle,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              style: TextStyle(color: Theme.of(context).hintColor),
            ),
        ],
      ),
    );

    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      child: InkWell(
        borderRadius: BorderRadius.circular(12),
        onTap: _openInSpotify,
        child: Card(
          child: Padding(
            padding: const EdgeInsets.all(12),
            child: Row(
              children: [
                rankBox,
                image,
                const SizedBox(width: 12),
                title,
                const SizedBox(width: 8),
                Tooltip(
                  message: 'Open in Spotify',
                  child: Icon(
                    Icons.open_in_new,
                    size: 18,
                    color: Theme.of(context).hintColor,
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }

  Widget _placeholderBox(BuildContext context, {double size = 44}) {
    return Container(
      width: size,
      height: size,
      decoration: BoxDecoration(
        color: Theme.of(context).colorScheme.surfaceContainerHighest,
        borderRadius: BorderRadius.circular(6),
      ),
      child: Icon(
        Icons.image,
        size: size * 0.5,
        color: Theme.of(context).hintColor,
      ),
    );
  }

  Future<void> _openInSpotify() async {
    final artistId =
        (item['spotifyArtistId'] ?? item['spotify_artist_id']) as String?;
    final trackId =
        (item['spotifyTrackId'] ?? item['spotify_track_id']) as String?;
    Uri? appUri;
    Uri? webUri;
    if (basis == 'tracks' && trackId != null && trackId.isNotEmpty) {
      appUri = Uri.parse('spotify:track:$trackId');
      webUri = Uri.https('open.spotify.com', '/track/$trackId');
    } else if (artistId != null && artistId.isNotEmpty) {
      appUri = Uri.parse('spotify:artist:$artistId');
      webUri = Uri.https('open.spotify.com', '/artist/$artistId');
    }
    if (appUri != null) {
      try {
        final ok = await launchUrl(
          appUri,
          mode: LaunchMode.externalApplication,
        );
        if (ok) return;
      } catch (_) {}
    }
    if (webUri != null) {
      await launchUrl(webUri, mode: LaunchMode.externalApplication);
    }
  }
}
