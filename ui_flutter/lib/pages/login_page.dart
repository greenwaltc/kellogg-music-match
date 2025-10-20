import 'package:flutter/material.dart';

import '../services/api_client.dart';
import '../services/auth_service.dart';
import 'package:shared_preferences/shared_preferences.dart';

class LoginPage extends StatefulWidget {
  const LoginPage({super.key, required this.onAuthenticated});
  final VoidCallback onAuthenticated;

  @override
  State<LoginPage> createState() => _LoginPageState();
}

class _LoginPageState extends State<LoginPage> {
  final _formKey = GlobalKey<FormState>();
  final _username = TextEditingController();
  final _password = TextEditingController();
  bool _submitting = false;
  String? _error;
  bool _showPassword = false;
  Map<String, List<String>> _fieldErrors = {};

  @override
  void initState() {
    super.initState();
    // Rebuild on input changes so the Sign in button enables immediately while typing
    void onChanged() => setState(() {});
    _username.addListener(onChanged);
    _password.addListener(onChanged);
  }

  @override
  void dispose() {
    _username.dispose();
    _password.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    if (!_formKey.currentState!.validate()) return;
    setState(() {
      _submitting = true;
      _error = null;
      _fieldErrors = {};
    });
    try {
      final prefs = await SharedPreferences.getInstance();
      final auth = AuthService(ApiClient(), prefs);
      await auth.login(username: _username.text.trim(), password: _password.text);
      if (!mounted) return;
      widget.onAuthenticated();
    } on ApiException catch (e) {
      Map<String, List<String>> fe = {};
      final details = e.details;
      if (details != null) {
        final errs = (details['errors'] ?? details['fieldErrors']);
        if (errs is Map) {
          errs.forEach((key, val) {
            if (key is String) {
              if (val is List) {
                fe[key] = val.whereType<String>().toList();
              } else if (val is String) {
                fe[key] = [val];
              }
            }
          });
        }
      }
      setState(() {
        _error = e.message;
        _fieldErrors = fe;
      });
    } catch (e) {
      setState(() => _error = 'Login failed');
    } finally {
      if (mounted) setState(() => _submitting = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final canSubmit = !_submitting && _username.text.trim().isNotEmpty && _password.text.isNotEmpty;
    return Center(
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 480),
        child: Card(
          margin: const EdgeInsets.all(16),
          child: Padding(
            padding: const EdgeInsets.all(16),
            child: Form(
              key: _formKey,
              child: Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  if (_error != null) ...[
                    Text(_error!, style: TextStyle(color: Theme.of(context).colorScheme.error)),
                    const SizedBox(height: 8),
                  ],
                  TextFormField(
                    controller: _username,
                    decoration: const InputDecoration(labelText: 'Username'),
                    validator: (v) {
                      if (v == null || v.trim().isEmpty) return 'Username required';
                      final msgs = _fieldErrors['username'];
                      if (msgs != null && msgs.isNotEmpty) return msgs.join('; ');
                      return null;
                    },
                  ),
                  const SizedBox(height: 12),
                  TextFormField(
                    controller: _password,
                    decoration: InputDecoration(
                      labelText: 'Password',
                      suffixIcon: IconButton(
                        onPressed: () => setState(() => _showPassword = !_showPassword),
                        icon: Icon(_showPassword ? Icons.visibility_off : Icons.visibility),
                      ),
                    ),
                    obscureText: !_showPassword,
                    validator: (v) {
                      if (v == null || v.isEmpty) return 'Password required';
                      final msgs = _fieldErrors['password'];
                      if (msgs != null && msgs.isNotEmpty) return msgs.join('; ');
                      return null;
                    },
                  ),
                  const SizedBox(height: 16),
                  FilledButton(
                    onPressed: canSubmit ? _submit : null,
                    child: _submitting
                        ? const SizedBox(width: 18, height: 18, child: CircularProgressIndicator(strokeWidth: 2))
                        : const Text('Sign in'),
                  ),
                  const SizedBox(height: 8),
                  TextButton(
                    onPressed: _submitting
                        ? null
                        : () => Navigator.of(context).pushReplacementNamed('/register'),
                    child: const Text('Create an account'),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}
