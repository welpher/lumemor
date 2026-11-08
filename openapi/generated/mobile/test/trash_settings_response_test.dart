import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for TrashSettingsResponse
void main() {
  final instance = TrashSettingsResponseBuilder();
  // TODO add properties to the builder and call build()

  group(TrashSettingsResponse, () {
    // The retention window of a soft-deleted asset in whole days, 0 to 365. A fresh instance (or a corrupt row) answers the default 30; 0 means the next expiry run purges everything trashed. 
    // int retentionDays
    test('to test the property `retentionDays`', () async {
      // TODO
    });

  });
}
