import 'package:flutter/material.dart';
import 'package:firebase_core/firebase_core.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:flutter/foundation.dart';

import 'pages/login_page.dart';
import 'pages/register_page.dart';
import 'services/api_client.dart';
import 'services/auth_service.dart';
import 'services/push_opt_in.dart';
import 'pages/debug_page.dart';
import 'theme/app_theme.dart';
import 'pages/spotify_connect_prompt.dart';
import 'services/spotify_service.dart';
import 'pages/matches_page.dart';

// Must be a top-level function for background handling
import 'package:firebase_messaging/firebase_messaging.dart';

@pragma('vm:entry-point')
Future<void> _firebaseMessagingBackgroundHandler(RemoteMessage message) async {
  // Ensure Firebase is initialized for background isolates
  try {
    await Firebase.initializeApp();
  } catch (_) {}
}

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  try {
    await Firebase.initializeApp();
    FirebaseMessaging.onBackgroundMessage(_firebaseMessagingBackgroundHandler);
  } catch (_) {
    // Continue without Firebase if not configured
  }
  runApp(const MainApp());
}

class MainApp extends StatelessWidget {
  const MainApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Kellogg Music Match',
      theme: AppTheme.light,
      darkTheme: AppTheme.dark,
      themeMode: ThemeMode.system,
      routes: {
        '/login': (_) => RootScaffold(
          body: LoginPage(
            onAuthenticated: () => _nav.currentState!.pushReplacementNamed('/'),
          ),
        ),
        '/register': (_) => RootScaffold(
          body: RegisterPage(
            onAuthenticated: () => _nav.currentState!.pushReplacementNamed('/'),
          ),
        ),
        if (!kReleaseMode)
          '/_debug': (_) => const DebugPage(), // only in debug/profile builds
      },
      navigatorKey: _nav,
      home: const AuthGate(child: HomePage()),
    );
  }
}

final GlobalKey<NavigatorState> _nav = GlobalKey<NavigatorState>();

class AuthGate extends StatefulWidget {
  const AuthGate({super.key, required this.child});
  final Widget child;

  @override
  State<AuthGate> createState() => _AuthGateState();
}

class _AuthGateState extends State<AuthGate> {
  bool _loading = true;
  bool _loggedIn = false;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    final prefs = await SharedPreferences.getInstance();
    final auth = AuthService(ApiClient(), prefs);
    setState(() {
      _loggedIn = auth.isLoggedIn;
      _loading = false;
    });
    if (auth.isLoggedIn) {
      // Fire-and-forget: prompt for push if needed
      final push = PushOptInManager(ApiClient(), prefs);
      // Delayed to allow build to settle
      Future.microtask(push.promptIfNeeded);
    }
  }

  @override
  Widget build(BuildContext context) {
    if (_loading) {
      return const RootScaffold(
        body: Center(child: CircularProgressIndicator()),
      );
    }
    if (!_loggedIn) {
      return RootScaffold(
        body: LoginPage(
          onAuthenticated: () async {
            final prefs = await SharedPreferences.getInstance();
            setState(() => _loggedIn = true);
            // Prompt for notifications after login
            final push = PushOptInManager(ApiClient(), prefs);
            Future.microtask(push.promptIfNeeded);
          },
        ),
      );
    }
    // For logged-in users, show the HomePage shell with bottom navigation
    return HomePage(onLogout: _handleLogout);
  }

  Future<void> _handleLogout() async {
    final prefs = await SharedPreferences.getInstance();
    final auth = AuthService(ApiClient(), prefs);
    await auth.logout();
    if (!mounted) return;
    setState(() => _loggedIn = false);
  }
}

class HomePage extends StatefulWidget {
  const HomePage({super.key, this.onLogout});
  final VoidCallback? onLogout;

  @override
  State<HomePage> createState() => _HomePageState();
}

class _HomePageState extends State<HomePage> {
  Map<String, dynamic>? _user;
  bool? _spotifyReady;
  bool _loadingStatus = true;
  int _currentIndex = 0;
  bool _routedForSpotifyMissing = false;

  @override
  void initState() {
    super.initState();
    _loadUser();
    _loadSpotifyStatus();
  }

  Future<void> _loadUser() async {
    final prefs = await SharedPreferences.getInstance();
    final auth = AuthService(ApiClient(), prefs);
    setState(() => _user = auth.currentUser);
  }

  Future<void> _loadSpotifyStatus() async {
    final prefs = await SharedPreferences.getInstance();
    final svc = SpotifyService(ApiClient(), prefs);
    try {
      final status = await svc.getStatus();
      setState(() {
        _spotifyReady = status['ready'] as bool? ?? false;
        _loadingStatus = false;
      });
      // If Spotify isn't connected, route to the Spotify tab once on initial load
      if ((_spotifyReady == false) && !_routedForSpotifyMissing) {
        setState(() {
          _currentIndex = 1; // Spotify tab
          _routedForSpotifyMissing = true;
        });
      }
    } catch (_) {
      setState(() {
        _spotifyReady = false;
        _loadingStatus = false;
      });
      if (!_routedForSpotifyMissing) {
        setState(() {
          _currentIndex = 1;
          _routedForSpotifyMissing = true;
        });
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final body = _buildBody();
    final bottom = BottomNavigationBar(
      currentIndex: _currentIndex,
      onTap: (i) => setState(() => _currentIndex = i),
      items: const [
        BottomNavigationBarItem(icon: Icon(Icons.group), label: 'Matches'),
        BottomNavigationBarItem(icon: Icon(Icons.graphic_eq), label: 'Spotify'),
        BottomNavigationBarItem(
          icon: Icon(Icons.settings_outlined),
          label: 'Settings',
        ),
      ],
    );
    return RootScaffold(
      body: body,
      onLogout: widget.onLogout,
      bottomNavigationBar: bottom,
    );
  }

  Widget _buildBody() {
    if (_user == null || _loadingStatus) {
      return const Center(child: CircularProgressIndicator());
    }
    switch (_currentIndex) {
      case 0: // Matches
        if (_spotifyReady == false) {
          return ListView(
            padding: const EdgeInsets.symmetric(vertical: 24),
            children: [
              Center(
                child: Column(
                  children: [
                    const Icon(Icons.group, size: 48),
                    const SizedBox(height: 12),
                    const Text(
                      'To find music matches, first connect your Spotify account.',
                      textAlign: TextAlign.center,
                    ),
                    const SizedBox(height: 8),
                    Text(
                      'Welcome, ${_user!['firstName']} ${_user!['lastName']}',
                    ),
                  ],
                ),
              ),
              // Reuse the same connect prompt component for consistency
              SpotifyConnectPrompt(onConnected: _loadSpotifyStatus),
            ],
          );
        }
        return const MatchesPage();
      case 1: // Spotify stats
        if (_spotifyReady == false) {
          return ListView(
            padding: const EdgeInsets.symmetric(vertical: 24),
            children: [
              Center(
                child: Column(
                  children: [
                    Text(
                      'Welcome, ${_user!['firstName']} ${_user!['lastName']}',
                    ),
                    const SizedBox(height: 8),
                    const Text(
                      'Connect Spotify to get personalized matches and concerts.',
                    ),
                  ],
                ),
              ),
              SpotifyConnectPrompt(onConnected: _loadSpotifyStatus),
            ],
          );
        }
        return Center(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: const [
              Icon(Icons.graphic_eq, size: 48),
              SizedBox(height: 12),
              Text('My Spotify stats (coming soon)'),
            ],
          ),
        );
      case 2: // Settings
      default:
        return ListView(
          padding: const EdgeInsets.all(16),
          children: [
            ListTile(
              leading: const Icon(Icons.account_circle_outlined),
              title: Text('${_user!['firstName']} ${_user!['lastName']}'),
              subtitle: Text('@${_user!['username']}'),
            ),
            const Divider(),
            ListTile(
              leading: const Icon(Icons.logout),
              title: const Text('Logout'),
              onTap: widget.onLogout,
            ),
            const SizedBox(height: 24),
            const Center(child: Text('Settings (coming soon)')),
          ],
        );
    }
  }
}

class RootScaffold extends StatelessWidget {
  const RootScaffold({
    super.key,
    required this.body,
    this.onLogout,
    this.bottomNavigationBar,
  });
  final Widget body;
  final VoidCallback? onLogout;
  final Widget? bottomNavigationBar;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: !kReleaseMode
            ? GestureDetector(
                onLongPress: () => Navigator.of(context).pushNamed('/_debug'),
                child: Row(
                  children: [
                    Image.asset('assets/icons/icon-192x192.png', height: 24),
                    const SizedBox(width: 8),
                    const Text('Kellogg Music Match'),
                  ],
                ),
              )
            : Row(
                children: [
                  Image.asset('assets/icons/icon-192x192.png', height: 24),
                  const SizedBox(width: 8),
                  const Text('Kellogg Music Match'),
                ],
              ),
        actions: [
          if (onLogout != null)
            IconButton(
              onPressed: onLogout,
              tooltip: 'Logout',
              icon: const Icon(Icons.logout),
            ),
        ],
      ),
      body: body,
      bottomNavigationBar: bottomNavigationBar,
    );
  }
}
