import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'package:ui_flutter/services/api_client.dart';
import 'package:ui_flutter/services/matching_service.dart';
import 'package:ui_flutter/pages/matches_page.dart';

class TestMatchingService extends MatchingService {
  TestMatchingService(super.api, super.prefs);

  @override
  Future<List<dynamic>> fetchMatches({
    required String basis,
    required String range,
    String? userName,
    int? limit,
    int? overlapsLimit,
    bool forceRefresh = false,
    bool includeDetails = false,
  }) async {
    // Return one match with known overlaps and top items
    return [
      {
        'name': 'Alice',
        'overlaps': [
          {'name': 'Radiohead'},
        ],
        'overlap': 1,
        'topArtists': [
          {'name': 'Radiohead', 'rank': 2},
          {'name': 'Muse', 'rank': 3},
        ],
        'topTracks': [
          {
            'name': 'Song A',
            'rank': 1,
            'artistNames': ['Artist A'],
          },
          {
            'name': 'Song B',
            'rank': 2,
            'artistNames': ['Artist B'],
          },
        ],
      },
    ];
  }
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  setUp(() async {
    SharedPreferences.setMockInitialValues({});
  });

  testWidgets('expands and shows top items; filters overlaps only', (
    tester,
  ) async {
    final prefs = await SharedPreferences.getInstance();
    final svc = TestMatchingService(
      ApiClient(baseUrl: 'http://localhost'),
      prefs,
    );

    await tester.pumpWidget(MaterialApp(home: MatchesPage(service: svc)));

    // Tap Find Matches
    expect(find.text('Find Matches'), findsOneWidget);
    await tester.tap(find.text('Find Matches'));
    await tester.pumpAndSettle();

    // Expect one result tile with name
    expect(find.text('Alice'), findsOneWidget);

    // Expand
    await tester.tap(find.text('Alice'));
    await tester.pumpAndSettle();

    // Should show both top artist items initially (preserves original casing)
    expect(find.text('Radiohead'), findsOneWidget);
    expect(find.text('Muse'), findsOneWidget);

    // Toggle only overlaps
    final switches = find.byType(Switch);
    expect(switches, findsOneWidget);
    await tester.tap(switches);
    await tester.pumpAndSettle();

    // Now only the overlap item remains
    expect(find.text('Radiohead'), findsOneWidget);
    expect(find.text('Muse'), findsNothing);
  });
}
