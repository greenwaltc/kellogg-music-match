import 'dart:async';
import 'package:shared_preferences/shared_preferences.dart';

import 'api_client.dart';
import 'auth_service.dart';

class MatchingService {
  MatchingService(this._api, this._prefs);

  final ApiClient _api;
  final SharedPreferences _prefs;

  // Simple in-memory cache for results across navigation
  static final Map<String, List<dynamic>> _cache = {};
  static String _key({
    required String basis,
    required String range,
    String? userName,
    int? limit,
    int? overlapsLimit,
    bool includeDetails = false,
  }) {
    final un = (userName ?? '').trim().toLowerCase();
    final lim = limit?.toString() ?? '';
    final ov = overlapsLimit?.toString() ?? '';
    final det = includeDetails ? '1' : '0';
    return 'basis:$basis|range:$range|q:$un|lim:$lim|ov:$ov|d:$det';
  }

  Future<String?> _token() async {
    final auth = AuthService(_api, _prefs);
    return await auth.getToken();
  }

  Future<List<dynamic>> fetchMatches({
    required String basis, // 'artists' | 'tracks'
    required String range, // 'short_term' | 'medium_term' | 'long_term'
    String? userName,
    int? limit,
    int? overlapsLimit,
    bool forceRefresh = false,
    bool includeDetails = false,
  }) async {
    final key = _key(
      basis: basis,
      range: range,
      userName: userName,
      limit: limit,
      overlapsLimit: overlapsLimit,
      includeDetails: includeDetails,
    );
    if (!forceRefresh && _cache.containsKey(key)) {
      return _cache[key]!;
    }
    final qp = <String, String>{
      'range': range,
      'basis': basis,
      'includeDetails': includeDetails ? 'true' : 'false',
    };
    if (userName != null && userName.trim().isNotEmpty) {
      qp['userName'] = userName.trim();
    }
    if (limit != null && limit > 0) {
      qp['limit'] = '$limit';
    }
    if (overlapsLimit != null && overlapsLimit > 0) {
      qp['overlapsLimit'] = '$overlapsLimit';
    }

    final path = Uri(path: '/findMusicMatches', queryParameters: qp).toString();
    final resp = await _api.postJsonAny(path, {
      // Backend ignores manual list in Spotify mode; send empty to satisfy body
      'artists': <String>[],
    }, bearerToken: await _token());

    // Expect an array of MatchUser
    final list = (resp as dynamic);
    // Some backends may return the array directly or wrap it; try both
    List<dynamic> result;
    if (list is List) {
      result = list;
    } else if (list is Map && list['matches'] is List) {
      result = (list['matches'] as List).cast<dynamic>();
    } else {
      // Fallback: treat unknown as empty
      result = <dynamic>[];
    }
    _cache[key] = result;
    return result;
  }

  static void clearCache() => _cache.clear();
}
