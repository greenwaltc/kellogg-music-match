import 'package:flutter/material.dart';
import 'package:firebase_core/firebase_core.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'pages/login_page.dart';
import 'pages/register_page.dart';
import 'services/api_client.dart';
import 'services/auth_service.dart';
import 'services/push_opt_in.dart';
import 'pages/debug_page.dart';

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
      theme: ThemeData(
        colorScheme: ColorScheme.fromSeed(seedColor: Colors.green),
        useMaterial3: true,
      ),
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
        // Hidden debug route: manually navigate via DevTools or deep link
        '/_debug': (_) => const DebugPage(),
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
    return RootScaffold(body: widget.child, onLogout: _handleLogout);
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
  const HomePage({super.key});

  @override
  State<HomePage> createState() => _HomePageState();
}

class _HomePageState extends State<HomePage> {
  Map<String, dynamic>? _user;

  @override
  void initState() {
    super.initState();
    _loadUser();
  }

  Future<void> _loadUser() async {
    final prefs = await SharedPreferences.getInstance();
    final auth = AuthService(ApiClient(), prefs);
    setState(() => _user = auth.currentUser);
  }

  @override
  Widget build(BuildContext context) {
    return Center(
      child: _user == null
          ? const CircularProgressIndicator()
          : Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                Text('Welcome, ${_user!['firstName']} ${_user!['lastName']}'),
                const SizedBox(height: 8),
                Text('Username: ${_user!['username']}'),
              ],
            ),
    );
  }
}

class RootScaffold extends StatelessWidget {
  const RootScaffold({super.key, required this.body, this.onLogout});
  final Widget body;
  final VoidCallback? onLogout;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: GestureDetector(
          onLongPress: () => Navigator.of(context).pushNamed('/_debug'),
          child: Row(
            children: [
              Image.asset('assets/icons/icon-192x192.png', height: 24),
              const SizedBox(width: 8),
              const Text('Kellogg Music Match'),
            ],
          ),
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
    );
  }
}
