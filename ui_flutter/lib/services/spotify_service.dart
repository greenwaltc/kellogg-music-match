import 'dart:convert';
import 'dart:math';

import 'package:flutter/foundation.dart';
import 'package:flutter_web_auth_2/flutter_web_auth_2.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:crypto/crypto.dart' as crypto;

import 'api_client.dart';
import 'auth_service.dart';

class SpotifyService {
  SpotifyService(this._api, this._prefs);

  final ApiClient _api;
  final SharedPreferences _prefs;

  static const _stateKey = 'spotify_auth_state';
  static const _verifierKey = 'spotify_code_verifier';
  static const _redirectKey = 'spotify_redirect_uri';

  // In Angular, redirectUri is window.origin + '/spotify/callback'.
  // For Flutter, use a custom scheme and have the backend accept it, or use a HTTPS callback that deep-links back.
  // We'll use an app scheme: affyne://spotify/callback and register an intent filter on Android later if needed.
  static const String callbackScheme = 'affyne';
  static const String callbackHost = 'spotify';
  static const String callbackPath = '/callback';
  static String get callbackUri =>
      '$callbackScheme://$callbackHost$callbackPath';

  Future<void> beginAuth({
    required String clientId,
    required String redirectUri,
    required List<String> scope,
  }) async {
    final state = _randomString(16);
    final codeVerifier = _randomString(64);
    await _prefs.setString(_stateKey, state);
    await _prefs.setString(_verifierKey, codeVerifier);
    await _prefs.setString(_redirectKey, redirectUri);
    final codeChallenge = _base64UrlNoPad(_sha256Utf8(codeVerifier));
    final params = {
      'response_type': 'code',
      'client_id': clientId,
      'scope': scope.join(' '),
      'redirect_uri': redirectUri,
      'state': state,
      'code_challenge_method': 'S256',
      'code_challenge': codeChallenge,
    };
    final authUrl = Uri.https(
      'accounts.spotify.com',
      '/authorize',
      params,
    ).toString();
    if (kDebugMode) {
      // ignore: avoid_print
      print('[Spotify] Redirecting to $authUrl');
    }
    // Launch external auth and wait for callback to our app scheme
    final cbScheme = Uri.parse(redirectUri).scheme.isNotEmpty
        ? Uri.parse(redirectUri).scheme
        : callbackScheme;
    final res = await FlutterWebAuth2.authenticate(
      url: authUrl,
      callbackUrlScheme: cbScheme,
    );
    final uri = Uri.parse(res);
    final code = uri.queryParameters['code'];
    final returnedState = uri.queryParameters['state'];
    final error = uri.queryParameters['error'];
    if (error != null) {
      throw Exception('Spotify authorization failed: $error');
    }
    if (code == null || code.isEmpty) {
      throw Exception('Missing authorization code');
    }
    final storedState = _prefs.getString(_stateKey);
    if (storedState != null && storedState != returnedState) {
      throw Exception('State mismatch');
    }
    final storedVerifier = _prefs.getString(_verifierKey) ?? '';
    final usedRedirect = _prefs.getString(_redirectKey) ?? redirectUri;
    await _exchangeCode(
      code,
      returnedState ?? '',
      storedVerifier,
      usedRedirect,
    );
  }

  Future<void> _exchangeCode(
    String code,
    String state,
    String codeVerifier,
    String redirectUri,
  ) async {
    try {
      await _api.postJson('/sync/spotify', {
        'code': code,
        'state': state,
        'code_verifier': codeVerifier,
        'redirect_uri': redirectUri,
      }, bearerToken: await _getToken());
      // Clear stored artifacts
      await _prefs.remove(_stateKey);
      await _prefs.remove(_verifierKey);
      await _prefs.remove(_redirectKey);
    } catch (e) {
      rethrow;
    }
  }

  Future<Map<String, dynamic>> getStatus() async {
    return await _api.getJson(
      '/sync/spotify/status',
      bearerToken: await _getToken(),
    );
  }

  Future<String?> _getToken() async {
    final auth = AuthService(_api, _prefs);
    return await auth.getToken();
  }

  static String _randomString(int length) {
    const chars =
        'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
    final rnd = Random.secure();
    final codeUnits = List.generate(
      length,
      (_) => chars.codeUnitAt(rnd.nextInt(chars.length)),
    );
    return String.fromCharCodes(codeUnits);
  }

  static List<int> _sha256Utf8(String input) {
    final bytes = utf8.encode(input);
    final digest = crypto.sha256.convert(bytes);
    return digest.bytes;
  }

  static String _base64UrlNoPad(List<int> bytes) {
    return base64UrlEncode(bytes).replaceAll('=', '');
  }
}
