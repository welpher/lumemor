import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for UpdateCheckSettingsResponse
void main() {
  final instance = UpdateCheckSettingsResponseBuilder();
  // TODO add properties to the builder and call build()

  group(UpdateCheckSettingsResponse, () {
    // The switch in force at the settings layer. `false` stops every scheduled and manual check; the cached answer is kept, so switching it back on resumes from the same state. 
    // bool enabled
    test('to test the property `enabled`', () async {
      // TODO
    });

    // The release feed URL the server probes: a JSON document whose newest tag rides `tag` (or `tag_name`). The server fetches it with a bounded timeout, through the environment's forward proxy, refusing internal addresses and never following a redirect. 
    // String endpoint
    test('to test the property `endpoint`', () async {
      // TODO
    });

    // The cadence in hours, 1-168: the server probes at most once per interval and caches the answer until the next one. 
    // int intervalHours
    test('to test the property `intervalHours`', () async {
      // TODO
    });

  });
}
