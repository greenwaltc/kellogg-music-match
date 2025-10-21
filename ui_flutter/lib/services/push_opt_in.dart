import 'dart:io' show Platform;

import 'package:device_info_plus/device_info_plus.dart';
import 'package:flutter/foundation.dart';
import 'package:package_info_plus/package_info_plus.dart';
import 'package:firebase_messaging/firebase_messaging.dart';
import 'package:permission_handler/permission_handler.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'api_client.dart';
import 'auth_service.dart';

/// Manages prompting the user for push notifications and registering the device token with the backend.
///
/// Token retrieval is stubbed for now; once Firebase/APNs integration is added, wire the real token.
class PushOptInManager {
  PushOptInManager(this._api, this._prefs);

  static const _prefKeyDecision =
      'push_opt_in_decision'; // values: 'granted' | 'denied'

  final ApiClient _api;
  final SharedPreferences _prefs;

  // Compile-time feature flags to enable/disable platform push prompting.
  // Defaults: iOS enabled (true), Android disabled (false) since we are not using Firebase in the client yet.
  static const bool _iosPushEnabled =
      String.fromEnvironment('IOS_PUSH_ENABLED', defaultValue: 'true') ==
      'true';
  static const bool _androidPushEnabled =
      String.fromEnvironment('ANDROID_PUSH_ENABLED', defaultValue: 'false') ==
      'true';
  static const bool _allowPseudoTokens =
      String.fromEnvironment(
        'ALLOW_PSEUDO_PUSH_TOKENS',
        defaultValue: 'false',
      ) ==
      'true';

  /// Call on app start or after successful login/register (when authenticated).
  Future<void> promptIfNeeded() async {
    // Only supported on native mobile targets
    if (kIsWeb ||
        (defaultTargetPlatform != TargetPlatform.iOS &&
            defaultTargetPlatform != TargetPlatform.android)) {
      return;
    }
    final isIOS = defaultTargetPlatform == TargetPlatform.iOS;
    final isAndroid = defaultTargetPlatform == TargetPlatform.android;
    // Respect feature flags per platform
    if ((isIOS && !_iosPushEnabled) || (isAndroid && !_androidPushEnabled)) {
      return;
    }
    // Only proceed for logged-in users; we need user context for backend registration.
    final auth = AuthService(_api, _prefs);
    if (!auth.isLoggedIn) return;

    final decided = _prefs.getString(_prefKeyDecision);
    if (decided == 'granted' || decided == 'denied') return;

    // iOS requires explicit request; Android 13+ also requires runtime permission.
    final status = await Permission.notification.status;
    if (status.isGranted) {
      await _prefs.setString(_prefKeyDecision, 'granted');
      await _registerDeviceToken();
      return;
    }

    // Ask the OS to prompt.
    final req = await Permission.notification.request();
    if (req.isGranted) {
      await _prefs.setString(_prefKeyDecision, 'granted');
      await _registerDeviceToken();
    } else {
      await _prefs.setString(_prefKeyDecision, 'denied');
    }
  }

  Future<void> _registerDeviceToken() async {
    // Gather basic device/app metadata
    final deviceInfo = DeviceInfoPlugin();
    String platform;
    String? bundleId;
    String? appPackage;
    String? deviceModel;
    String? osVersion;
    String? appVersion;

    final pkg = await PackageInfo.fromPlatform();
    appVersion = '${pkg.version}+${pkg.buildNumber}';

    if (defaultTargetPlatform == TargetPlatform.iOS) {
      platform = 'ios';
      final info = await deviceInfo.iosInfo;
      deviceModel = info.utsname.machine;
      osVersion = info.systemVersion;
      bundleId = pkg.packageName; // On iOS, packageName is the bundle ID
    } else if (defaultTargetPlatform == TargetPlatform.android) {
      platform = 'android';
      final info = await deviceInfo.androidInfo;
      deviceModel = info.model;
      osVersion = info.version.release;
      appPackage = pkg.packageName;
    } else {
      // Not a supported native platform; skip
      return;
    }

    // Respect feature flags before registering token as well.
    if ((platform == 'ios' && !_iosPushEnabled) ||
        (platform == 'android' && !_androidPushEnabled)) {
      return;
    }

    // Try to obtain a real push token when platform + feature flags allow
    String token = '';
    try {
      if (platform == 'android') {
        // Firebase Messaging for Android
        final fcm = FirebaseMessaging.instance;
        await fcm.setAutoInitEnabled(true);
        // On Android 13+ permission was requested earlier; ensure granted
        final settings = await fcm.getNotificationSettings();
        if (kDebugMode) {
          // ignore: avoid_print
          print(
            '[PushOptIn] Android notification settings: ${settings.authorizationStatus}',
          );
        }
        if (settings.authorizationStatus == AuthorizationStatus.authorized ||
            settings.authorizationStatus == AuthorizationStatus.provisional) {
          token = await _tryGetFcmTokenWithRetry(fcm);
          if (kDebugMode) {
            // ignore: avoid_print
            print(
              '[PushOptIn] FCM token acquired (android): ${token.isNotEmpty}',
            );
          }
          // Listen for refresh and update backend silently
          FirebaseMessaging.instance.onTokenRefresh.listen((newToken) async {
            try {
              await _api.registerDeviceToken(
                platform: platform,
                token: newToken,
                bundleId: bundleId,
                appPackage: appPackage,
                deviceModel: deviceModel,
                osVersion: osVersion,
                appVersion: appVersion,
                bearerToken: AuthService(_api, _prefs).token,
              );
            } catch (_) {}
          });
        }
      } else if (platform == 'ios') {
        // iOS will also use Firebase if configured; else, APNs direct token would be needed
        final fcm = FirebaseMessaging.instance;
        await fcm.setAutoInitEnabled(true);
        final apsToken = await fcm.getAPNSToken();
        // Ensure permissions via previous prompt
        final settings = await fcm.getNotificationSettings();
        if (kDebugMode) {
          // ignore: avoid_print
          print(
            '[PushOptIn] iOS notification settings: ${settings.authorizationStatus}, apns token present: ${apsToken != null && apsToken.isNotEmpty}',
          );
        }
        if ((apsToken != null && apsToken.isNotEmpty) &&
            (settings.authorizationStatus == AuthorizationStatus.authorized ||
                settings.authorizationStatus ==
                    AuthorizationStatus.provisional)) {
          token = await _tryGetFcmTokenWithRetry(fcm);
          if (kDebugMode) {
            // ignore: avoid_print
            print('[PushOptIn] FCM token acquired (ios): ${token.isNotEmpty}');
          }
          FirebaseMessaging.instance.onTokenRefresh.listen((newToken) async {
            try {
              await _api.registerDeviceToken(
                platform: platform,
                token: newToken,
                bundleId: bundleId,
                appPackage: appPackage,
                deviceModel: deviceModel,
                osVersion: osVersion,
                appVersion: appVersion,
                bearerToken: AuthService(_api, _prefs).token,
              );
            } catch (_) {}
          });
        }
      }
    } catch (e) {
      if (kDebugMode) {
        // ignore: avoid_print
        print('[PushOptIn] Firebase token retrieval failed: $e');
      }
    }

    // Fallback for dev if token is unavailable and pseudo tokens are allowed
    if (token.isEmpty && _allowPseudoTokens) {
      token = await _derivePseudoToken(deviceModel, osVersion, appVersion);
    }

    try {
      await _api.registerDeviceToken(
        platform: platform,
        token: token,
        bundleId: bundleId,
        appPackage: appPackage,
        deviceModel: deviceModel,
        osVersion: osVersion,
        appVersion: appVersion,
        bearerToken: AuthService(_api, _prefs).token,
      );
    } catch (e) {
      if (kDebugMode) {
        // ignore: avoid_print
        print('[PushOptIn] registerDeviceToken failed: $e');
      }
    }
  }

  Future<String> _derivePseudoToken(
    String? model,
    String? os,
    String? appVer,
  ) async {
    // DO NOT use in production; this is just to plumb the flow until FCM/APNs is added.
    final base = [
      Platform.operatingSystem,
      model ?? '',
      os ?? '',
      appVer ?? '',
    ].join('|');
    // Cheap hash
    return base.codeUnits
        .fold<int>(0, (a, b) => (a * 31 + b) & 0x7fffffff)
        .toRadixString(16);
  }

  // Attempts to get an FCM token with simple backoff retries to smooth over transient
  // Firebase Installations outages on emulators or flaky networks.
  Future<String> _tryGetFcmTokenWithRetry(FirebaseMessaging fcm) async {
    const maxAttempts = 3;
    const delays = [
      Duration(milliseconds: 500),
      Duration(seconds: 2),
      Duration(seconds: 5),
    ];
    for (var i = 0; i < maxAttempts; i++) {
      try {
        final t = await fcm.getToken();
        if (t != null && t.isNotEmpty) return t;
      } catch (e) {
        if (kDebugMode) {
          // ignore: avoid_print
          print('[PushOptIn] getToken attempt ${i + 1} failed: $e');
        }
      }
      await Future.delayed(delays[i]);
    }
    return '';
  }
}
