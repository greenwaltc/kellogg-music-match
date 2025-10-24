import 'package:flutter/material.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../services/api_client.dart';
import '../services/auth_service.dart';
import '../services/spotify_service.dart';

class SpotifyConnectPrompt extends StatefulWidget {
  const SpotifyConnectPrompt({super.key, this.onConnected});
  final VoidCallback? onConnected;

  @override
  State<SpotifyConnectPrompt> createState() => _SpotifyConnectPromptState();
}

class _SpotifyConnectPromptState extends State<SpotifyConnectPrompt> {
  bool _loading = false;
  String? _error;

  Future<void> _connect() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final prefs = await SharedPreferences.getInstance();
      // Prefer values from dart-define; fallback to scheme callback
      final svc = SpotifyService(ApiClient(), prefs);
      const clientId = String.fromEnvironment(
        'SPOTIFY_CLIENT_ID',
        defaultValue: '',
      );
      const redirectDefine = String.fromEnvironment('SPOTIFY_REDIRECT_URI');
      final redirectUri = redirectDefine.isNotEmpty
          ? redirectDefine
          : SpotifyService.callbackUri;
      if (clientId.isEmpty) {
        throw Exception(
          'Missing SPOTIFY_CLIENT_ID. Provide via --dart-define=SPOTIFY_CLIENT_ID=...',
        );
      }
      final scopes = const [
        'user-read-email',
        'user-read-private',
        'user-top-read',
        'playlist-read-private',
      ];
      await svc.beginAuth(
        clientId: clientId,
        redirectUri: redirectUri,
        scope: scopes,
      );
      if (mounted) {
        // Navigate to syncing screen and await completion
        final result = await Navigator.of(context).push<bool>(
          MaterialPageRoute(builder: (_) => const SpotifySyncingPage()),
        );
        if (result == true) {
          // Refresh parent state so HomePage hides the connect prompt immediately
          widget.onConnected?.call();
        }
      }
    } catch (e) {
      setState(() {
        _error = e.toString();
      });
    } finally {
      if (mounted) {
        setState(() {
          _loading = false;
        });
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    return Card(
      margin: const EdgeInsets.all(16),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text(
              'Connect your Spotify',
              style: TextStyle(fontSize: 18, fontWeight: FontWeight.w600),
            ),
            const SizedBox(height: 8),
            const Text(
              'To find musically similar classmates and tailor concerts to you, please connect your Spotify account.',
            ),
            const SizedBox(height: 12),
            if (_error != null)
              Text(_error!, style: const TextStyle(color: Colors.red)),
            const SizedBox(height: 8),
            ElevatedButton.icon(
              onPressed: _loading ? null : _connect,
              icon: const Icon(Icons.headphones),
              label: Text(_loading ? 'Connecting…' : 'Connect / Sync Spotify'),
            ),
          ],
        ),
      ),
    );
  }
}

class SpotifySyncingPage extends StatefulWidget {
  const SpotifySyncingPage({super.key});
  @override
  State<SpotifySyncingPage> createState() => _SpotifySyncingPageState();
}

class _SpotifySyncingPageState extends State<SpotifySyncingPage> {
  bool _loading = true;
  String? _status;
  int? _progress;
  String? _message;
  String? _error;
  bool? _ready;
  bool _popped = false;

  @override
  void initState() {
    super.initState();
    _poll();
  }

  Future<void> _poll([int attempt = 0]) async {
    try {
      final prefs = await SharedPreferences.getInstance();
      final res = await ApiClient().getJson(
        '/sync/spotify/status',
        bearerToken: await AuthService(ApiClient(), prefs).getToken(),
      );
      setState(() {
        _status = res['status'] as String?;
        _progress = res['progress'] as int?;
        _message = res['message'] as String?;
        _ready = res['ready'] as bool?;
        _loading =
            _status != 'complete' &&
            _status != 'failed' &&
            _status != 'cancelled';
      });
      if (!_loading && mounted && !_popped) {
        if (_status == 'complete' && (_ready == true)) {
          _popped = true;
          Navigator.of(context).pop(true);
          return;
        }
      }
      if (_loading) {
        await Future.delayed(const Duration(seconds: 1));
        if (mounted) _poll(attempt + 1);
      }
    } catch (e) {
      setState(() {
        _error = e.toString();
        _loading = false;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Spotify Authorization')),
      body: Center(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              if (_loading) const CircularProgressIndicator(),
              const SizedBox(height: 12),
              if (_loading)
                Text(
                  'Syncing your Spotify data${_progress != null ? '… $_progress%' : '…'}',
                ),
              if (!_loading && _status == 'complete') const Text('Done.'),
              if (_error != null)
                Text(
                  'Error: $_error',
                  style: const TextStyle(color: Colors.red),
                ),
              if (!_loading && _status != 'complete' && _status != null)
                Text(
                  'Status: $_status${_message != null ? ' — $_message' : ''}',
                ),
            ],
          ),
        ),
      ),
    );
  }
}
