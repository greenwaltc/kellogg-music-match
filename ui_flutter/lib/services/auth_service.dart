import 'dart:convert';

import 'package:shared_preferences/shared_preferences.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

import 'api_client.dart';

class AuthService {
  AuthService(this._api, this._prefs);

  static const _userKey = 'auth_user';
  static const _tokenKey = 'auth_token';
  static const _secureTokenKey = 'auth_token_secure';
  static const _secure = FlutterSecureStorage();

  final ApiClient _api;
  final SharedPreferences _prefs;

  Map<String, dynamic>? get currentUser {
    final raw = _prefs.getString(_userKey);
    if (raw == null) return null;
    return jsonDecode(raw) as Map<String, dynamic>;
  }

  // Back-compat: synchronous access used throughout the app; reads SharedPreferences copy
  String? get token => _prefs.getString(_tokenKey);

  // Hardened access: prefer secure storage (Keychain/Keystore) when available
  Future<String?> getToken() async {
    final secure = await _secure.read(key: _secureTokenKey);
    if (secure != null && secure.isNotEmpty) return secure;
    return _prefs.getString(_tokenKey);
  }

  bool get isLoggedIn => currentUser != null;

  Future<void> logout() async {
    await _prefs.remove(_userKey);
    await _prefs.remove(_tokenKey);
    await _secure.delete(key: _secureTokenKey);
  }

  Future<Map<String, dynamic>> login({
    required String username,
    required String password,
  }) async {
    final resp = await _api.postJson('/login', {
      'username': username,
      'password': password,
    });
    await _saveAuth(resp);
    return resp;
  }

  Future<Map<String, dynamic>> register({
    required String username,
    required String email,
    required String firstName,
    required String lastName,
    required String password,
    required String program,
    required int graduationYear,
  }) async {
    final resp = await _api.postJson('/register', {
      'username': username,
      'email': email,
      'firstName': firstName,
      'lastName': lastName,
      'password': password,
      'program': program,
      'graduationYear': graduationYear,
    });
    await _saveAuth(resp);
    return resp;
  }

  Future<void> _saveAuth(Map<String, dynamic> authResponse) async {
    final user = authResponse['user'];
    final token = authResponse['token'];
    if (user is Map<String, dynamic>) {
      await _prefs.setString(_userKey, jsonEncode(user));
    }
    if (token is String) {
      await _secure.write(key: _secureTokenKey, value: token);
      await _prefs.setString(_tokenKey, token);
    } else {
      await _prefs.remove(_tokenKey);
      await _secure.delete(key: _secureTokenKey);
    }
  }
}
