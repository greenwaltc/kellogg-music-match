import 'package:flutter/material.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:url_launcher/url_launcher.dart';

import '../services/api_client.dart';
import '../services/matching_service.dart';
// auth not needed directly here

class MatchesPage extends StatefulWidget {
  const MatchesPage({super.key, this.service});

  final MatchingService? service;

  @override
  State<MatchesPage> createState() => _MatchesPageState();
}

class _MatchesPageState extends State<MatchesPage> {
  late final ApiClient _api;
  SharedPreferences? _prefs;
  MatchingService? _svc;
  final ScrollController _scrollController = ScrollController();

  String _basis = 'artists'; // 'artists' | 'tracks'
  String _range = 'medium_term'; // short_term | medium_term | long_term
  final TextEditingController _searchCtrl = TextEditingController();

  bool _loading = false;
  String? _error;
  List<dynamic> _matches = const [];
  final Set<int> _loadingDetails = {};

  // Expanded state and per-card toggle (all vs overlaps only)
  final Set<int> _expanded = {};
  final Map<int, bool> _onlyOverlap = {};
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
    final savedBasis = prefs.getString('matches_basis');
    final savedRange = prefs.getString('matches_range');
    final savedQuery = prefs.getString('matches_query');
    setState(() {
      _api = ApiClient();
      _prefs = prefs;
      _svc = widget.service ?? MatchingService(_api, prefs);
      if (savedBasis == 'artists' || savedBasis == 'tracks') {
        _basis = savedBasis!;
      }
      if (savedRange == 'short_term' ||
          savedRange == 'medium_term' ||
          savedRange == 'long_term') {
        _range = savedRange!;
      }
      if (savedQuery != null) {
        _searchCtrl.text = savedQuery;
      }
    });
  }

  @override
  void dispose() {
    _searchCtrl.dispose();
    _scrollController.removeListener(_onScroll);
    _scrollController.dispose();
    super.dispose();
  }

  Future<void> _submit({bool forceRefresh = false}) async {
    if (_svc == null) return;
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final list = await _svc!.fetchMatches(
        basis: _basis,
        range: _range,
        userName: _searchCtrl.text.trim().isEmpty
            ? null
            : _searchCtrl.text.trim(),
        limit: 20,
        overlapsLimit: 0,
        forceRefresh: forceRefresh,
      );
      setState(() {
        _matches = list;
      });
    } catch (e) {
      setState(() {
        _error = e.toString();
      });
    } finally {
      setState(() {
        _loading = false;
      });
    }
  }

  Future<void> _maybeLoadDetailsForIndex(int index) async {
    if (_svc == null) return;
    if (index < 0 || index >= _matches.length) return;
    final item = _matches[index] as Map<String, dynamic>;
    // If we already have overlaps or top lists, no need to load
    final hasOverlaps =
        item['overlaps'] is List && (item['overlaps'] as List).isNotEmpty;
    final hasTopArtists =
        item['topArtists'] is List && (item['topArtists'] as List).isNotEmpty;
    final hasTopTracks =
        item['topTracks'] is List && (item['topTracks'] as List).isNotEmpty;
    if (hasOverlaps || hasTopArtists || hasTopTracks) return;
    final username = item['username'] as String?;
    if (username == null || username.trim().isEmpty) return;
    setState(() => _loadingDetails.add(index));
    try {
      final detailsList = await _svc!.fetchMatches(
        basis: _basis,
        range: _range,
        userName: username, // pass username via name filter
        includeDetails: true,
        limit: 1,
        overlapsLimit: null,
        forceRefresh: true, // ensure we bypass list cache for specific user
      );
      if (detailsList.isNotEmpty && detailsList.first is Map<String, dynamic>) {
        final details = (detailsList.first as Map<String, dynamic>);
        final merged = {...item, ...details};
        setState(() {
          _matches = List<dynamic>.from(_matches);
          _matches[index] = merged;
        });
      }
    } catch (e) {
      // Surface error but don't block UI
      setState(() {
        _error = e.toString();
      });
    } finally {
      if (mounted) {
        setState(() => _loadingDetails.remove(index));
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final topControls = _buildControls(context);
    final list = _buildResults(context);
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
        setState(() {
          _basis = s.first;
          _expanded.clear();
          _onlyOverlap.clear();
          _matches = const [];
        });
        _prefs?.setString('matches_basis', _basis);
        // Auto-refresh when basis changes
        _submit(forceRefresh: true);
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
        setState(() {
          _range = s.first;
          _expanded.clear();
          _onlyOverlap.clear();
          _matches = const [];
        });
        _prefs?.setString('matches_range', _range);
        // Auto-refresh when range changes
        _submit(forceRefresh: true);
      },
    );

    final search = TextField(
      controller: _searchCtrl,
      decoration: const InputDecoration(
        prefixIcon: Icon(Icons.search),
        labelText: 'Filter by name',
        hintText: 'Search first or last name',
        border: OutlineInputBorder(),
      ),
      textInputAction: TextInputAction.search,
      onSubmitted: (_) {
        _prefs?.setString('matches_query', _searchCtrl.text.trim());
        _submit();
      },
    );

    final submit = FilledButton.icon(
      onPressed: _loading
          ? null
          : () {
              _prefs?.setString('matches_query', _searchCtrl.text.trim());
              _submit();
            },
      icon: const Icon(Icons.playlist_add_check),
      label: const Text('Find Matches'),
    );

    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 12, 16, 12),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          basisToggle,
          const SizedBox(height: 8),
          rangeToggle,
          const SizedBox(height: 12),
          Row(
            children: [
              Expanded(child: search),
              const SizedBox(width: 12),
              submit,
            ],
          ),
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

  Widget _buildResults(BuildContext context) {
    if (_loading && _matches.isEmpty) {
      return const Center(child: CircularProgressIndicator());
    }
    return RefreshIndicator(
      onRefresh: () => _submit(forceRefresh: true),
      child: NotificationListener<ScrollNotification>(
        onNotification: (n) {
          final current = n.metrics.pixels > 200;
          if (current != _showScrollToTop) {
            setState(() => _showScrollToTop = current);
          }
          return false;
        },
        child: ListView.builder(
          controller: _scrollController,
          padding: const EdgeInsets.only(top: 8, bottom: 16),
          itemCount: _matches.length,
          itemBuilder: (context, index) {
            final item = _matches[index] as Map<String, dynamic>;
            return _MatchUserTile(
              index: index,
              data: item,
              basis: _basis,
              expanded: _expanded.contains(index),
              loadingDetails: _loadingDetails.contains(index),
              onlyOverlap: _onlyOverlap[index] ?? false,
              onToggleExpanded: () {
                final willExpand = !_expanded.contains(index);
                setState(() {
                  if (willExpand) {
                    _expanded.add(index);
                  } else {
                    _expanded.remove(index);
                  }
                });
                if (willExpand) {
                  _maybeLoadDetailsForIndex(index);
                }
              },
              onToggleOnlyOverlap: (val) {
                setState(() => _onlyOverlap[index] = val);
              },
            );
          },
        ),
      ),
    );
  }
}

class _MatchUserTile extends StatelessWidget {
  const _MatchUserTile({
    required this.index,
    required this.data,
    required this.basis,
    required this.expanded,
    required this.loadingDetails,
    required this.onlyOverlap,
    required this.onToggleExpanded,
    required this.onToggleOnlyOverlap,
  });

  final int index;
  final Map<String, dynamic> data;
  final String basis;
  final bool expanded;
  final bool loadingDetails;
  final bool onlyOverlap;
  final VoidCallback onToggleExpanded;
  final ValueChanged<bool> onToggleOnlyOverlap;

  @override
  Widget build(BuildContext context) {
    final name = data['name'] as String? ?? '';
    final overlapsList =
        (data['overlaps'] as List?)?.cast<dynamic>() ?? const [];
    // Prefer the explicit overlap property from the API, fall back to overlaps length
    final overlap = (data['overlap'] as num?)?.toInt() ?? overlapsList.length;

    final tile = ListTile(
      leading: CircleAvatar(child: Text('${index + 1}')),
      title: Text(name),
      subtitle: Text('Overlap: $overlap'),
      trailing: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          if (overlap > 0)
            Container(
              padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
              decoration: BoxDecoration(
                color: Theme.of(context).colorScheme.primaryContainer,
                borderRadius: BorderRadius.circular(12),
              ),
              child: Text(
                '$overlap',
                style: TextStyle(
                  color: Theme.of(context).colorScheme.onPrimaryContainer,
                ),
              ),
            ),
          const SizedBox(width: 8),
          // While loading per-user details, show a small spinner in the header
          if (loadingDetails)
            const SizedBox(
              width: 18,
              height: 18,
              child: CircularProgressIndicator(strokeWidth: 2),
            )
          else
            const Icon(Icons.expand_more),
        ],
      ),
      onTap: onToggleExpanded,
    );

    if (!expanded) {
      return Padding(
        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
        child: Card(child: tile),
      );
    }

    // Expanded view content
    final overlaps = _overlapNameSet(data);
    final allItems = _topItems(data, basis);
    final shownItems = onlyOverlap
        ? allItems.where((e) => overlaps.contains(_itemName(e))).toList()
        : allItems;

    final toggle = Row(
      mainAxisAlignment: MainAxisAlignment.end,
      children: [
        const Text('Show only overlaps'),
        Switch(value: onlyOverlap, onChanged: onToggleOnlyOverlap),
      ],
    );

    final list = SizedBox(
      height: 180,
      child: Scrollbar(
        child: ListView.separated(
          itemCount: shownItems.length,
          separatorBuilder: (_, __) => const Divider(height: 1),
          itemBuilder: (context, i) {
            final it = shownItems[i] as Map<String, dynamic>;
            final displayName = (it['name'] as String?) ?? '';
            final nm = _itemName(it); // lowercase for comparison
            final rk = (it['rank'] as num?)?.toInt();
            final isOverlap = overlaps.contains(nm);
            return ListTile(
              dense: true,
              leading: CircleAvatar(
                radius: 14,
                backgroundColor: isOverlap
                    ? Theme.of(context).colorScheme.primaryContainer
                    : Theme.of(context).colorScheme.surfaceVariant,
                child: Text(
                  rk != null ? '$rk' : '•',
                  style: const TextStyle(fontSize: 12),
                ),
              ),
              title: Text(displayName),
              subtitle: basis == 'tracks' && it['artistNames'] is List
                  ? Text(
                      (it['artistNames'] as List).cast<String>().join(', '),
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                    )
                  : null,
              trailing: Tooltip(
                message: 'Open in Spotify',
                child: Icon(
                  Icons.open_in_new,
                  size: 16,
                  color: Theme.of(context).hintColor,
                ),
              ),
              onTap: () => _openInSpotify(it),
            );
          },
        ),
      ),
    );

    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      child: Card(
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            tile,
            // Show a thin progress bar while details are loading (regardless of initial content)
            if (loadingDetails)
              const Padding(
                padding: EdgeInsets.symmetric(horizontal: 12, vertical: 8),
                child: LinearProgressIndicator(minHeight: 2),
              ),
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 12),
              child: toggle,
            ),
            Padding(
              padding: const EdgeInsets.fromLTRB(12, 0, 12, 12),
              child: list,
            ),
            if (shownItems.isEmpty)
              const Padding(
                padding: EdgeInsets.only(bottom: 12),
                child: Center(child: Text('No top items available')),
              ),
          ],
        ),
      ),
    );
  }

  static Set<String> _overlapNameSet(Map<String, dynamic> data) {
    final overlaps = (data['overlaps'] as List?)?.cast<dynamic>() ?? const [];
    final names = <String>{};
    for (final ov in overlaps) {
      if (ov is Map && ov['name'] is String) {
        names.add((ov['name'] as String).toLowerCase());
      }
    }
    return names;
  }

  static List<dynamic> _topItems(Map<String, dynamic> data, String basis) {
    // Prefer field matching the current basis, but fall back to whichever exists
    if (basis == 'tracks') {
      final vt = data['topTracks'];
      if (vt is List) return vt;
      final va = data['topArtists'];
      if (va is List) return va;
    } else {
      final va = data['topArtists'];
      if (va is List) return va;
      final vt = data['topTracks'];
      if (vt is List) return vt;
    }
    return const [];
  }

  static String _itemName(Map<String, dynamic> it) {
    final nm = it['name'];
    if (nm is String) return nm.toLowerCase();
    return '';
  }

  static Future<void> _openInSpotify(Map<String, dynamic> it) async {
    // Prefer track link when available, otherwise artist
    final trackId = (it['spotifyTrackId'] ?? it['spotify_track_id']) as String?;
    final artistId =
        (it['spotifyArtistId'] ?? it['spotify_artist_id']) as String?;
    Uri? appUri;
    Uri? webUri;
    if (trackId != null && trackId.isNotEmpty) {
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
