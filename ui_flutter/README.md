# ui_flutter

## Push notifications opt-in (dev)

We purposefully avoid using Firebase in the client during early development to minimize push costs and setup overhead. By default:
- iOS push prompting/registration is ENABLED (uses a pseudo token for now).
- Android push prompting/registration is DISABLED (no Firebase on the client yet).

You can adjust this behavior at build time with dart-defines (see below). We use a temporary pseudo token until APNs/FCM is wired.

### After adding new plugins

If you see MissingPluginException for permission_handler / device_info_plus / package_info_plus after a hot reload, do a full rebuild so native code is integrated:

```
flutter clean
flutter pub get
flutter run
```

### iOS

- Ensure CocoaPods is installed and run from the project root: `flutter pub get` then `flutter run`.
- The runtime permission dialog is handled by permission_handler; no extra Info.plist entries are required for notifications permission text.

### Android

- Android 13+ requires POST_NOTIFICATIONS. This is declared in `android/app/src/main/AndroidManifest.xml`.
- The app requests `Permission.notification` at runtime.
- Push prompting/registration is disabled by default on Android via a compile-time flag.

#### Build-time flags (dart-define)
- `IOS_PUSH_ENABLED` (default: `true`)
- `ANDROID_PUSH_ENABLED` (default: `false`)

Examples:

```bash
# Run with default behavior (iOS on, Android off)
flutter run

# Enable Android push prompting/registration too
flutter run --dart-define=ANDROID_PUSH_ENABLED=true

# Disable iOS push prompting temporarily
flutter run --dart-define=IOS_PUSH_ENABLED=false
```

### Next steps for real push tokens

You can keep developing without Firebase/APNs (current build uses a pseudo token in dev). When you’re ready for production push, wire up real tokens using Firebase Cloud Messaging (FCM). This works for both Android and iOS (FCM will bridge to APNs on iOS when you upload your APNs key to Firebase).

Option A (recommended): Use FCM tokens on both Android and iOS
- Simpler client code (one token type), server can send via FCM for both platforms.

Option B: Use FCM for Android and APNs device tokens for iOS
- Requires retrieving APNs tokens directly (extra plugin/code). Only choose this if your backend must send to APNs directly and not via FCM on iOS.

Below are steps for Option A (FCM on both platforms). If you prefer Option B, see the note at the end.

#### 1) Add dependencies (optional until you’re ready)
- Add to `pubspec.yaml` dependencies:
	- `firebase_core`
	- `firebase_messaging`
	- Optional (foreground banners on Android/iOS): `flutter_local_notifications`

#### 2) Configure Firebase (FlutterFire CLI)
Using the CLI is the easiest way to set up both platforms. This modifies native Gradle/Xcode config for you.

Commands (run from `ui_flutter/`):

```bash
# Install once if you don't already have it
dart pub global activate flutterfire_cli

# Configure Firebase for this app; select your Firebase project, iOS bundle ID, and Android package
flutterfire configure
```

What you’ll need:
- iOS bundle identifier (e.g., com.yourorg.kelloggMusicMatch)
- Android applicationId (package name) (e.g., com.yourorg.kellogg_music_match)
- A Firebase project

This will place `GoogleService-Info.plist` in `ios/Runner/` and `google-services.json` in `android/app/`, and apply required build plugins.

#### 3) iOS-specific setup
- In Xcode, open `ios/Runner.xcworkspace`:
	- Signing & Capabilities: add “Push Notifications” and “Background Modes” with “Remote notifications”.
	- Ensure a valid Team is selected.
- In the Firebase Console, upload your APNs Auth Key (or certificate) so FCM can deliver to APNs.
- No custom Info.plist changes for permission text are needed for notifications.

#### 4) Android-specific setup
- `flutterfire configure` adds the Google services plugin. Verify:
	- `android/build.gradle` has `classpath 'com.google.gms:google-services:...'
	- `android/app/build.gradle` applies `com.google.gms.google-services`.
- We already declare `POST_NOTIFICATIONS` (Android 13+). Keep the runtime prompt via `permission_handler`.

#### 5) Initialize Firebase in Dart
- In `lib/main.dart` before `runApp`, initialize Firebase once:

```dart
import 'package:firebase_core/firebase_core.dart';

void main() async {
	WidgetsFlutterBinding.ensureInitialized();
	await Firebase.initializeApp();
	runApp(const App());
}
```

If you used `flutterfire configure`, it will also generate `firebase_options.dart`. Import and pass options if needed.

#### 6) Replace pseudo-token with a real token in `services/push_opt_in.dart`
- Keep the current OS permission flow. After permission is granted, obtain an FCM token and register it:

```dart
import 'package:firebase_messaging/firebase_messaging.dart';

Future<void> _registerDeviceToken() async {
	// ...collect platform + device/app metadata as already implemented...

	// Ensure permission is granted (we already prompted via permission_handler)
	// Optionally, also ask FCM to ensure its internal permission state is synced
	await FirebaseMessaging.instance.requestPermission();

	// Get the FCM registration token (works on Android and iOS when APNs is set up in Firebase)
	final fcmToken = await FirebaseMessaging.instance.getToken();
	if (fcmToken == null || fcmToken.isEmpty) {
		if (kDebugMode) {
			print('[PushOptIn] No FCM token available');
		}
		return;
	}

	try {
		await _api.registerDeviceToken(
			platform: defaultTargetPlatform == TargetPlatform.iOS ? 'ios' : 'android',
			token: fcmToken, // REAL token now
			bundleId: bundleId,
			appPackage: appPackage,
			deviceModel: deviceModel,
			osVersion: osVersion,
			appVersion: appVersion,
			bearerToken: AuthService(_api, _prefs).token,
		);
	} catch (e) {
		if (kDebugMode) {
			print('[PushOptIn] registerDeviceToken failed: $e');
		}
	}
}

void startPushTokenRefreshListener() {
	FirebaseMessaging.instance.onTokenRefresh.listen((newToken) async {
		try {
			await _api.registerDeviceToken(
				platform: defaultTargetPlatform == TargetPlatform.iOS ? 'ios' : 'android',
				token: newToken,
				bundleId: bundleId, // reuse device/app metadata or re-fetch as needed
				appPackage: appPackage,
				deviceModel: deviceModel,
				osVersion: osVersion,
				appVersion: appVersion,
				bearerToken: AuthService(_api, _prefs).token,
			);
		} catch (e) {
			if (kDebugMode) {
				print('[PushOptIn] token refresh register failed: $e');
			}
		}
	});
}
```

Call `startPushTokenRefreshListener()` once after login/app start (when authenticated) so token changes are captured.

Notes:
- On iOS, FCM requires the APNs Auth Key to be uploaded in Firebase to obtain a valid FCM token and deliver notifications.
- On some devices, token is only available after first successful run with granted permission; retry on next launch is expected.

#### 7) Foreground presentation (optional)
If you want heads-up banners while the app is foregrounded, wire `flutter_local_notifications` and/or iOS notification center delegate. This isn’t required to retrieve/register tokens.

#### 8) Verify end-to-end
- Do a clean rebuild after adding Firebase plugins (hot reload isn’t enough for native changes):

```bash
flutter clean
flutter pub get
flutter run
```

- Sign in, accept the notification prompt, and inspect backend logs for a call to `/push/device/register` with a non-empty token. Use your backend’s test-send endpoint to push a test.

---

Option B: Use APNs tokens directly on iOS
- If your backend must send via APNs directly, you can retrieve an APNs device token on iOS and send that instead of an FCM token. You’ll need a plugin that exposes the APNs token (for example, via native bridging or a dedicated APNs plugin). Then set `platform = 'ios'` and `token = <apns-device-token>` in `registerDeviceToken`.
- You will still need to enable iOS capabilities (Push Notifications + Background Modes) and handle token refresh by re-registering on app start or when the plugin signals a new token.
