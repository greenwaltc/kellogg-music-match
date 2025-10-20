import 'package:flutter/material.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'dart:math' as math;

import '../services/api_client.dart';
import '../services/auth_service.dart';

class RegisterPage extends StatefulWidget {
  const RegisterPage({super.key, required this.onAuthenticated});
  final VoidCallback onAuthenticated;

  @override
  State<RegisterPage> createState() => _RegisterPageState();
}

class _RegisterPageState extends State<RegisterPage> {
  final _formKey = GlobalKey<FormState>();
  final _username = TextEditingController();
  final _email = TextEditingController();
  final _firstName = TextEditingController();
  final _lastName = TextEditingController();
  final _password = TextEditingController();
  final _confirmPassword = TextEditingController();
  final _program = ValueNotifier<String>('2Y');
  final _graduationYear = TextEditingController();

  bool _submitting = false;
  String? _error;
  Map<String, List<String>> _fieldErrors = {};

  @override
  void dispose() {
    _username.dispose();
    _email.dispose();
    _firstName.dispose();
    _lastName.dispose();
    _password.dispose();
  _confirmPassword.dispose();
    _graduationYear.dispose();
    _program.dispose();
    super.dispose();
  }

  @override
  void initState() {
    super.initState();
    // Live button gating updates while typing
    void onChanged() => setState(() {});
    _username.addListener(onChanged);
    _email.addListener(onChanged);
    _firstName.addListener(onChanged);
    _lastName.addListener(onChanged);
    _password.addListener(onChanged);
    _confirmPassword.addListener(onChanged);
    _graduationYear.addListener(onChanged);
  }

  // Password complexity state
  bool _lenOk = false;
  bool _upperOk = false;
  bool _lowerOk = false;
  bool _digitOk = false;
  bool _matchOk = false;
  bool _showPassword = false;
  bool _showConfirm = false;

  double _estimateEntropy(String s) {
    if (s.isEmpty) return 0;
    int pool = 0;
    if (RegExp(r'[a-z]').hasMatch(s)) pool += 26;
    if (RegExp(r'[A-Z]').hasMatch(s)) pool += 26;
    if (RegExp(r'[0-9]').hasMatch(s)) pool += 10;
    if (RegExp(r'[^A-Za-z0-9]').hasMatch(s)) pool += 33; // approx printable symbols
    if (pool == 0) return 0;
    // entropy ≈ length * log2(pool)
    final entropy = s.length * (math.log(pool) / math.log(2));
    return entropy;
  }

  String _entropyLabel(double bits) {
    if (bits < 28) return 'Weak';
    if (bits < 60) return 'Medium';
    return 'Strong';
  }

  Color _strengthColor(BuildContext ctx) {
    final label = _entropyLabel(_estimateEntropy(_password.text));
    switch (label) {
      case 'Weak':
        return Theme.of(ctx).colorScheme.error;
      case 'Medium':
        return Colors.orange.shade700;
      default:
        return Colors.green.shade700;
    }
  }

  void _onPasswordChanged(String value) {
    setState(() {
      _lenOk = value.length >= 8;
      _upperOk = value.contains(RegExp(r'[A-Z]'));
      _lowerOk = value.contains(RegExp(r'[a-z]'));
      _digitOk = value.contains(RegExp(r'[0-9]'));
      _matchOk = _confirmPassword.text == value && _confirmPassword.text.isNotEmpty;
    });
  }

  void _onConfirmChanged(String value) {
    setState(() {
      _matchOk = value == _password.text && value.isNotEmpty;
    });
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
      await auth.register(
        username: _username.text.trim(),
        email: _email.text.trim(),
        firstName: _firstName.text.trim(),
        lastName: _lastName.text.trim(),
        password: _password.text,
        program: _program.value,
        graduationYear: int.parse(_graduationYear.text.trim()),
      );
      if (!mounted) return;
      widget.onAuthenticated();
    } on ApiException catch (e) {
      Map<String, List<String>> fe = {};
      final details = e.details;
      if (details != null) {
        // Supported shapes:
        // { errors: { field: ['msg'] } } or { fieldErrors: { field: ['msg'] } }
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
      setState(() => _error = 'Registration failed');
    } finally {
      if (mounted) setState(() => _submitting = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Center(
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 560),
        child: Card(
          margin: const EdgeInsets.all(16),
          child: Padding(
            padding: const EdgeInsets.all(16),
            child: Form(
              key: _formKey,
              child: SingleChildScrollView(
                child: Column(
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
                        controller: _email,
                        decoration: const InputDecoration(labelText: 'Email'),
                        keyboardType: TextInputType.emailAddress,
                        validator: (v) {
                          final val = v?.trim() ?? '';
                          if (val.isEmpty) return 'Email required';
                          if (!val.contains('@')) return 'Invalid email';
                          final msgs = _fieldErrors['email'];
                          if (msgs != null && msgs.isNotEmpty) return msgs.join('; ');
                          return null;
                        },
                      ),
                      const SizedBox(height: 12),
                      Row(children: [
                        Expanded(
                          child: TextFormField(
                            controller: _firstName,
                            decoration: const InputDecoration(labelText: 'First name'),
                            validator: (v) {
                              if (v == null || v.trim().isEmpty) return 'First name required';
                              final msgs = _fieldErrors['firstName'];
                              if (msgs != null && msgs.isNotEmpty) return msgs.join('; ');
                              return null;
                            },
                          ),
                        ),
                        const SizedBox(width: 12),
                        Expanded(
                          child: TextFormField(
                            controller: _lastName,
                            decoration: const InputDecoration(labelText: 'Last name'),
                            validator: (v) {
                              if (v == null || v.trim().isEmpty) return 'Last name required';
                              final msgs = _fieldErrors['lastName'];
                              if (msgs != null && msgs.isNotEmpty) return msgs.join('; ');
                              return null;
                            },
                          ),
                        ),
                      ]),
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
                        onChanged: _onPasswordChanged,
                        validator: (v) {
                          final val = v ?? '';
                          if (val.isEmpty) return 'Password required';
                          if (val.length < 8) return 'Must be at least 8 characters';
                          if (!RegExp(r'[A-Z]').hasMatch(val)) return 'Must contain an uppercase letter';
                          if (!RegExp(r'[a-z]').hasMatch(val)) return 'Must contain a lowercase letter';
                          if (!RegExp(r'[0-9]').hasMatch(val)) return 'Must contain a digit';
                          final msgs = _fieldErrors['password'];
                          if (msgs != null && msgs.isNotEmpty) return msgs.join('; ');
                          return null;
                        },
                      ),
                      const SizedBox(height: 6),
                      Row(
                        children: [
                          Expanded(
                            child: LinearProgressIndicator(
                              value: (() {
                                final bits = _estimateEntropy(_password.text);
                                return (bits / 80).clamp(0.0, 1.0);
                              })(),
                              backgroundColor: Colors.grey.shade300,
                              color: _strengthColor(context),
                              minHeight: 6,
                            ),
                          ),
                          const SizedBox(width: 8),
                          Text(
                            _entropyLabel(_estimateEntropy(_password.text)),
                            style: TextStyle(fontWeight: FontWeight.w600, color: _strengthColor(context)),
                          ),
                        ],
                      ),
                      const SizedBox(height: 8),
                      _PasswordCriteria(
                        lenOk: _lenOk,
                        upperOk: _upperOk,
                        lowerOk: _lowerOk,
                        digitOk: _digitOk,
                        matchOk: _matchOk,
                      ),
                      const SizedBox(height: 12),
                      TextFormField(
                        controller: _confirmPassword,
                        decoration: InputDecoration(
                          labelText: 'Confirm password',
                          suffixIcon: IconButton(
                            onPressed: () => setState(() => _showConfirm = !_showConfirm),
                            icon: Icon(_showConfirm ? Icons.visibility_off : Icons.visibility),
                          ),
                        ),
                        obscureText: !_showConfirm,
                        onChanged: _onConfirmChanged,
                        validator: (v) {
                          final val = v ?? '';
                          if (val.isEmpty) return 'Please re-enter your password';
                          if (val != _password.text) return 'Passwords do not match';
                          final msgs = _fieldErrors['confirmPassword'];
                          if (msgs != null && msgs.isNotEmpty) return msgs.join('; ');
                          return null;
                        },
                      ),
                      const SizedBox(height: 12),
                      Row(children: [
                        Expanded(
                          child: ValueListenableBuilder<String>(
                            valueListenable: _program,
                            builder: (context, value, _) {
                              return DropdownButtonFormField<String>(
                                initialValue: value,
                                decoration: const InputDecoration(labelText: 'Program'),
                                items: const [
                                  DropdownMenuItem(value: '2Y', child: Text('2Y')),
                                  DropdownMenuItem(value: '1Y', child: Text('1Y')),
                                  DropdownMenuItem(value: 'MMM', child: Text('MMM')),
                                  DropdownMenuItem(value: 'MBAi', child: Text('MBAi')),
                                  DropdownMenuItem(value: 'JD-MBA', child: Text('JD-MBA')),
                                  DropdownMenuItem(value: 'MD-MBA', child: Text('MD-MBA')),
                                  DropdownMenuItem(value: 'EWMBA', child: Text('EWMBA')),
                                  DropdownMenuItem(value: 'JV', child: Text('JV')),
                                ],
                                onChanged: (v) => _program.value = v ?? '2Y',
                              );
                            },
                          ),
                        ),
                        const SizedBox(width: 12),
                        Expanded(
                          child: TextFormField(
                            controller: _graduationYear,
                            decoration: const InputDecoration(labelText: 'Graduation year'),
                            keyboardType: TextInputType.number,
                            validator: (v) {
                              final val = int.tryParse(v?.trim() ?? '');
                              if (val == null) return 'Enter a valid year';
                              if (val < 2024 || val > 2035) return 'Enter a reasonable year';
                              final msgs = _fieldErrors['graduationYear'];
                              if (msgs != null && msgs.isNotEmpty) return msgs.join('; ');
                              return null;
                            },
                          ),
                        ),
                      ]),
                      const SizedBox(height: 16),
                      FilledButton(
                        onPressed: _submitting || !(_lenOk && _upperOk && _lowerOk && _digitOk && _matchOk)
                            ? null
                            : _submit,
                        child: _submitting
                            ? const SizedBox(width: 18, height: 18, child: CircularProgressIndicator(strokeWidth: 2))
                            : const Text('Create account'),
                      ),
                      const SizedBox(height: 8),
                    TextButton(
                      onPressed: _submitting
                          ? null
                          : () => Navigator.of(context).pushReplacementNamed('/login'),
                      child: const Text('Already have an account? Sign in'),
                    ),
                  ],
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }
}

class _PasswordCriteria extends StatelessWidget {
  const _PasswordCriteria({
    required this.lenOk,
    required this.upperOk,
    required this.lowerOk,
    required this.digitOk,
    required this.matchOk,
  });

  final bool lenOk;
  final bool upperOk;
  final bool lowerOk;
  final bool digitOk;
  final bool matchOk;

  @override
  Widget build(BuildContext context) {
    final okColor = Colors.green.shade700;
    final badColor = Theme.of(context).colorScheme.error;
    Widget row(bool ok, String text) => Row(
          children: [
            Icon(ok ? Icons.check_circle : Icons.cancel, size: 16, color: ok ? okColor : badColor),
            const SizedBox(width: 6),
            Flexible(child: Text(text)),
          ],
        );

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        row(lenOk, 'At least 8 characters'),
        row(upperOk, 'Contains uppercase letter'),
        row(lowerOk, 'Contains lowercase letter'),
        row(digitOk, 'Contains a number'),
        row(matchOk, 'Passwords match'),
      ],
    );
  }
}
