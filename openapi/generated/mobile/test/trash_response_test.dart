import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for TrashResponse
void main() {
  final instance = TrashResponseBuilder();
  // TODO add properties to the builder and call build()

  group(TrashResponse, () {
    // The trashed assets of the caller, ordered by `deletedAt` descending. 
    // BuiltList<TrashItem> items
    test('to test the property `items`', () async {
      // TODO
    });

    // The instance's trash retention window in whole days, the same setting the listing's `remainingDays` countdowns were computed against (`trash_retention_days`, default 30). The trash page's note renders it; adjusting the setting stays the administrator's surface, but the number itself is visible to every signed-in member. A corrupt setting row answers the default 30 -- the same degradation the countdowns follow. 
    // int retentionDays
    test('to test the property `retentionDays`', () async {
      // TODO
    });

  });
}
