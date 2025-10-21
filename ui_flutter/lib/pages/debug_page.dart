import 'package:flutter/material.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:firebase_messaging/firebase_messaging.dart';

import '../services/api_client.dart';
import '../services/auth_service.dart';

class DebugPage extends StatefulWidget {
  const DebugPage({super.key});

  @override
  State<DebugPage> createState() => _DebugPageState();
}

class _DebugPageState extends State<DebugPage> {
  String? _fcmToken;
  String? _authUser;
  String? _error;
  bool _sending = false;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    try {
      final t = await FirebaseMessaging.instance.getToken();
      final prefs = await SharedPreferences.getInstance();
      final auth = AuthService(ApiClient(), prefs);
      setState(() {
        _fcmToken = t;
        _authUser = auth.currentUser != null
            ? auth.currentUser!['username'] as String?
            : null;
      });
    } catch (e) {
      setState(() => _error = e.toString());
    }
  }

  Future<void> _sendTestPush() async {
    setState(() {
      _sending = true;
      _error = null;
    });
    try {
      final prefs = await SharedPreferences.getInstance();
      final auth = AuthService(ApiClient(), prefs);
      final token = auth.token;
      if (token == null) throw Exception('No auth token');
      final api = ApiClient();
      await api.postJson('/push/test/enqueue', {
        'message': 'Hello from DebugPage',
        'target': 'native',
      }, bearerToken: token);
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(const SnackBar(content: Text('Test push enqueued')));
      }
    } catch (e) {
      setState(() => _error = e.toString());
    } finally {
      if (mounted) setState(() => _sending = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Debug')),
      body: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('User: ${_authUser ?? 'not logged in'}'),
            const SizedBox(height: 8),
            const Text('FCM token:'),
            SelectableText(_fcmToken ?? '(none)'),
            const SizedBox(height: 16),
            if (_error != null)
              Text(_error!, style: const TextStyle(color: Colors.red)),
            const Spacer(),
            Row(
              children: [
                ElevatedButton.icon(
                  onPressed: _sending ? null : _sendTestPush,
                  icon: const Icon(Icons.send),
                  label: const Text('Send test push'),
                ),
                const SizedBox(width: 12),
                OutlinedButton.icon(
                  onPressed: _sending ? null : _load,
                  icon: const Icon(Icons.refresh),
                  label: const Text('Refresh info'),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}
