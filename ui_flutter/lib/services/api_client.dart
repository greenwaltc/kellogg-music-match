import 'dart:convert';
import 'package:flutter/foundation.dart';
import 'package:http/http.dart' as http;

class ApiClient {
  ApiClient({http.Client? httpClient, String? baseUrl})
    : _http = httpClient ?? http.Client(),
      baseUrl = baseUrl ?? _defaultBaseUrl;

  final http.Client _http;
  final String baseUrl;

  Map<String, String> get _jsonHeaders => {
    'Content-Type': 'application/json',
    'Accept': 'application/json',
  };

  Future<Map<String, dynamic>> getJson(
    String path, {
    String? bearerToken,
  }) async {
    final uri = Uri.parse('$baseUrl$path');
    final headers = {..._jsonHeaders};
    if (bearerToken != null && bearerToken.isNotEmpty) {
      headers['Authorization'] = 'Bearer $bearerToken';
    }
    final resp = await _http.get(uri, headers: headers);
    if (resp.statusCode >= 200 && resp.statusCode < 300) {
      if (resp.body.isEmpty) return <String, dynamic>{};
      return jsonDecode(resp.body) as Map<String, dynamic>;
    }
    // Reuse error extraction logic by throwing ApiException similar to postJson
    String msg;
    try {
      final dyn = jsonDecode(resp.body);
      if (dyn is Map<String, dynamic>) {
        msg = (dyn['message'] as String?)?.trim() ?? 'Request failed';
        throw ApiException(
          resp.statusCode,
          msg.isNotEmpty ? msg : 'Request failed',
          details: dyn,
        );
      }
    } catch (_) {}
    msg = resp.body.trim();
    throw ApiException(
      resp.statusCode,
      msg.isNotEmpty ? msg : 'Request failed',
    );
  }

  Future<Map<String, dynamic>> postJson(
    String path,
    Map<String, dynamic> body, {
    String? bearerToken,
  }) async {
    final uri = Uri.parse('$baseUrl$path');
    final headers = {..._jsonHeaders};
    if (bearerToken != null && bearerToken.isNotEmpty) {
      headers['Authorization'] = 'Bearer $bearerToken';
    }
    final resp = await _http.post(
      uri,
      headers: headers,
      body: jsonEncode(body),
    );
    if (resp.statusCode >= 200 && resp.statusCode < 300) {
      if (resp.body.isEmpty) return <String, dynamic>{};
      return jsonDecode(resp.body) as Map<String, dynamic>;
    }
    // Try to parse and extract a helpful error message from varied shapes
    Map<String, dynamic>? decodedJson;
    String extractMessage(String body) {
      try {
        final dynamic decoded = jsonDecode(body);
        if (decoded is String) return decoded;
        if (decoded is Map<String, dynamic>) {
          decodedJson = decoded;
          // Common keys in our backend and common APIs
          final candidates = [
            decoded['message'],
            decoded['error_description'],
            decoded['error'],
            decoded['detail'],
            decoded['title'],
          ];
          for (final c in candidates) {
            if (c is String && c.trim().isNotEmpty) return c.trim();
          }
          // Handle { errors: ['msg1','msg2'] } or { errors: {field: ['msg'] } }
          final errors = decoded['errors'];
          if (errors is List) {
            final parts = errors
                .whereType<String>()
                .map((e) => e.trim())
                .where((e) => e.isNotEmpty)
                .toList();
            if (parts.isNotEmpty) return parts.join('; ');
          }
          if (errors is Map) {
            final parts = <String>[];
            errors.forEach((key, val) {
              if (val is List) {
                parts.addAll(val.whereType<String>().map((e) => e.trim()));
              } else if (val is String) {
                parts.add(val.trim());
              }
            });
            if (parts.isNotEmpty) return parts.join('; ');
          }
        }
      } catch (_) {
        // Not JSON; fall back below
      }
      final trimmed = body.trim();
      if (trimmed.isNotEmpty && !trimmed.startsWith('<')) {
        return trimmed.length > 500 ? '${trimmed.substring(0, 500)}…' : trimmed;
      }
      return 'Request failed';
    }

    final msg = extractMessage(resp.body);
    if (kDebugMode) {
      // ignore: avoid_print
      print('[ApiClient] POST $path failed ${resp.statusCode}: $msg');
    }
    // Attach details if we parsed JSON
    Map<String, dynamic>? details;
    try {
      if (decodedJson is Map<String, dynamic>) {
        details = decodedJson as Map<String, dynamic>;
      } else {
        final dyn = jsonDecode(resp.body);
        if (dyn is Map<String, dynamic>) details = dyn;
      }
    } catch (_) {}
    throw ApiException(
      resp.statusCode,
      msg.isNotEmpty ? msg : 'Request failed (${resp.statusCode})',
      details: details,
    );
  }

  // Same as postJson but returns dynamic to support endpoints that return arrays
  Future<dynamic> postJsonAny(
    String path,
    Map<String, dynamic> body, {
    String? bearerToken,
  }) async {
    final uri = Uri.parse('$baseUrl$path');
    final headers = {..._jsonHeaders};
    if (bearerToken != null && bearerToken.isNotEmpty) {
      headers['Authorization'] = 'Bearer $bearerToken';
    }
    final resp = await _http.post(
      uri,
      headers: headers,
      body: jsonEncode(body),
    );
    if (resp.statusCode >= 200 && resp.statusCode < 300) {
      if (resp.body.isEmpty) return null;
      return jsonDecode(resp.body);
    }
    // Reuse error extraction behavior
    String msg;
    Map<String, dynamic>? details;
    try {
      final dynamic decoded = jsonDecode(resp.body);
      if (decoded is Map<String, dynamic>) {
        details = decoded;
        final candidates = [
          decoded['message'],
          decoded['error_description'],
          decoded['error'],
          decoded['detail'],
          decoded['title'],
        ];
        final found = candidates.firstWhere(
          (e) => e is String && e.trim().isNotEmpty,
          orElse: () => null,
        );
        if (found is String) {
          msg = found.trim();
        } else {
          msg = 'Request failed';
        }
      } else if (decoded is String) {
        msg = decoded.trim();
      } else {
        msg = 'Request failed';
      }
    } catch (_) {
      final trimmed = resp.body.trim();
      if (trimmed.isNotEmpty && !trimmed.startsWith('<')) {
        msg = trimmed.length > 500 ? '${trimmed.substring(0, 500)}…' : trimmed;
      } else {
        msg = 'Request failed';
      }
    }
    if (kDebugMode) {
      // ignore: avoid_print
      print('[ApiClient] POST $path failed ${resp.statusCode}: $msg');
    }
    throw ApiException(
      resp.statusCode,
      msg.isNotEmpty ? msg : 'Request failed (${resp.statusCode})',
      details: details,
    );
  }

  // Convenience helper for device-token registration
  Future<void> registerDeviceToken({
    required String platform, // 'ios' | 'android'
    required String token,
    String? bundleId,
    String? appPackage,
    String? deviceModel,
    String? osVersion,
    String? appVersion,
    String? bearerToken,
  }) async {
    final payload = <String, dynamic>{'platform': platform, 'token': token};
    if (bundleId != null && bundleId.isNotEmpty) payload['bundleId'] = bundleId;
    if (appPackage != null && appPackage.isNotEmpty)
      payload['appPackage'] = appPackage;
    if (deviceModel != null && deviceModel.isNotEmpty)
      payload['deviceModel'] = deviceModel;
    if (osVersion != null && osVersion.isNotEmpty)
      payload['osVersion'] = osVersion;
    if (appVersion != null && appVersion.isNotEmpty)
      payload['appVersion'] = appVersion;
    await postJson('/push/device/register', payload, bearerToken: bearerToken);
  }

  Future<Map<String, dynamic>> deleteJson(
    String path, {
    String? bearerToken,
  }) async {
    final uri = Uri.parse('$baseUrl$path');
    final headers = {..._jsonHeaders};
    if (bearerToken != null && bearerToken.isNotEmpty) {
      headers['Authorization'] = 'Bearer $bearerToken';
    }
    final resp = await _http.delete(uri, headers: headers);
    if (resp.statusCode >= 200 && resp.statusCode < 300) {
      if (resp.body.isEmpty) return <String, dynamic>{};
      return jsonDecode(resp.body) as Map<String, dynamic>;
    }
    String msg;
    try {
      final dyn = jsonDecode(resp.body);
      if (dyn is Map<String, dynamic>) {
        msg = (dyn['message'] as String?)?.trim() ?? 'Request failed';
        throw ApiException(
          resp.statusCode,
          msg.isNotEmpty ? msg : 'Request failed',
          details: dyn,
        );
      }
    } catch (_) {}
    msg = resp.body.trim();
    throw ApiException(
      resp.statusCode,
      msg.isNotEmpty ? msg : 'Request failed',
    );
  }
}

class ApiException implements Exception {
  ApiException(this.status, this.message, {this.details});
  final int status;
  final String message;
  final Map<String, dynamic>? details;

  @override
  String toString() => 'ApiException($status): $message';
}

// Default base URL with Android emulator support.
// You can override with: --dart-define=API_BASE_URL=http://your-host:8080
String get _defaultBaseUrl {
  const define = String.fromEnvironment('API_BASE_URL');
  if (define.isNotEmpty) return define;
  // Prefer local dev endpoints first
  if (!kIsWeb && defaultTargetPlatform == TargetPlatform.android) {
    // Android emulator has a special alias to host loopback
    return 'http://10.0.2.2:8080';
  }
  return 'http://localhost:8080';
}
