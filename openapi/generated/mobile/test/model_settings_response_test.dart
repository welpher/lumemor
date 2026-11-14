import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for ModelSettingsResponse
void main() {
  final instance = ModelSettingsResponseBuilder();
  // TODO add properties to the builder and call build()

  group(ModelSettingsResponse, () {
    // Whether an AI job whose slot is configured but whose weights are missing may start the model group's download on its own. It never fills an unconfigured slot: the first-run download stays the administrator's own act. 
    // bool autoDownload
    test('to test the property `autoDownload`', () async {
      // TODO
    });

  });
}
