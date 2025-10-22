import 'dart:async';

import 'package:shared_preferences/shared_preferences.dart';

import 'api_client.dart';
import 'auth_service.dart';

class SpotifyTopService {
  SpotifyTopService(this._api, this._prefs);

  final ApiClient _api;
  final SharedPreferences _prefs;

  Future<String?> _token() async => AuthService(_api, _prefs).getToken();

  String? _userId() {
    final user = AuthService(_api, _prefs).currentUser;
    final id = user != null ? user['id'] as String? : null;
    return (id != null && id.isNotEmpty) ? id : null;
  }

  Future<Map<String, dynamic>> fetchTopArtistsPage({
    required String range, // 'short_term' | 'medium_term' | 'long_term'
    int limit = 20,
    int offset = 0,
  }) async {
    final uid = _userId();
    if (uid == null) throw StateError('Missing current user id');
    final qp = <String, String>{
      'range': range,
      'limit': '$limit',
      'offset': '$offset',
    };
    final path = Uri(
      path: '/users/$uid/top-artists',
      queryParameters: qp,
    ).toString();
    final resp = await _api.getJson(path, bearerToken: await _token());
    return resp; // { items, hasMore, totalCount }
  }

  Future<Map<String, dynamic>> fetchTopTracksPage({
    required String range,
    int limit = 20,
    int offset = 0,
  }) async {
    final uid = _userId();
    if (uid == null) throw StateError('Missing current user id');
    final qp = <String, String>{
      'range': range,
      'limit': '$limit',
      'offset': '$offset',
    };
    final path = Uri(
      path: '/users/$uid/top-tracks',
      queryParameters: qp,
    ).toString();
    final resp = await _api.getJson(path, bearerToken: await _token());
    return resp; // { items, hasMore, totalCount }
  }

  Future<void> refreshFromSpotify() async {
    await _api.postJson(
      '/sync/spotify/refresh',
      {},
      bearerToken: await _token(),
    );
  }

  Future<Map<String, dynamic>> getSpotifySyncStatus() async {
    return await _api.getJson(
      '/sync/spotify/status',
      bearerToken: await _token(),
    );
  }

  Future<void> cancelSpotifySync() async {
    await _api.deleteJson('/sync/spotify/retry', bearerToken: await _token());
  }
}
