import 'package:flutter/material.dart';

// Angular PWA tokens mapped into Flutter ThemeData
// Light palette
const _lightBg = Color(0xFFF6F5FA); // --color-bg
const _lightSurface = Color(0xFFFFFFFF); // --color-surface
const _lightBorder = Color(0xFFE2DFF0); // --color-border
const _lightText = Color(0xFF2D2640); // --color-text
const _lightAccent = Color(0xFF6A2AB9); // --color-accent
const _lightAccentAlt = Color(0xFF7E3FD6); // --color-accent-alt
const _lightError = Color(0xFFDC2626); // --color-error

// Dark palette
const _darkBg = Color(0xFF12101A);
const _darkSurface = Color(0xFF1F1B29);
const _darkBorder = Color(0xFF2F2A3D);
const _darkText = Color(0xFFECEAF3);
const _darkAccent = Color(0xFF6A2AB9);
const _darkAccentAlt = Color(0xFF9333EA);
const _darkError = Color(0xFFF87171);

class AppTheme {
  static ThemeData get light {
    final colorScheme = ColorScheme(
      brightness: Brightness.light,
      primary: _lightAccent,
      onPrimary: Colors.white,
      secondary: _lightAccentAlt,
      onSecondary: Colors.white,
      error: _lightError,
      onError: Colors.white,
      surface: _lightSurface,
      onSurface: _lightText,
      // Map a few M3 extras
      tertiary: _lightAccentAlt,
      onTertiary: Colors.white,
      outline: _lightBorder,
  shadow: Colors.black.withValues(alpha: 0.12),
      inverseSurface: _lightText,
      onInverseSurface: _lightSurface,
      inversePrimary: _lightAccentAlt,
    );

    return ThemeData(
      useMaterial3: true,
      colorScheme: colorScheme,
      scaffoldBackgroundColor: _lightBg,
      appBarTheme: AppBarTheme(
        backgroundColor: _lightSurface,
        foregroundColor: _lightText,
        elevation: 0,
        surfaceTintColor: Colors.transparent,
      ),
      dividerColor: _lightBorder,
      textTheme: const TextTheme().apply(
        bodyColor: _lightText,
        displayColor: _lightText,
      ),
    );
  }

  static ThemeData get dark {
    final colorScheme = ColorScheme(
      brightness: Brightness.dark,
      primary: _darkAccent,
      onPrimary: Colors.white,
      secondary: _darkAccentAlt,
      onSecondary: Colors.white,
      error: _darkError,
      onError: Colors.black,
      surface: _darkSurface,
      onSurface: _darkText,
      tertiary: _darkAccentAlt,
      onTertiary: Colors.white,
      outline: _darkBorder,
  shadow: Colors.black.withValues(alpha: 0.4),
      inverseSurface: _darkText,
      onInverseSurface: _darkSurface,
      inversePrimary: _darkAccentAlt,
    );

    return ThemeData(
      useMaterial3: true,
      colorScheme: colorScheme,
      scaffoldBackgroundColor: _darkBg,
      appBarTheme: AppBarTheme(
        backgroundColor: _darkSurface,
        foregroundColor: _darkText,
        elevation: 0,
        surfaceTintColor: Colors.transparent,
      ),
      dividerColor: _darkBorder,
      textTheme: const TextTheme().apply(
        bodyColor: _darkText,
        displayColor: _darkText,
      ),
    );
  }
}
