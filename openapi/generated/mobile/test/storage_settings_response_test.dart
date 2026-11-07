import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for StorageSettingsResponse
void main() {
  final instance = StorageSettingsResponseBuilder();
  // TODO add properties to the builder and call build()

  group(StorageSettingsResponse, () {
    // The template new uploads render their placement from. A fresh instance (or a corrupt row) answers the default `{{y}}/{{MM}}/{{filename}}`. 
    // String template
    test('to test the property `template`', () async {
      // TODO
    });

    // The template rendered over the fixed sample asset with the extension appended, as a stored relative path -- exactly the placement a real upload of the sample file would land at. 
    // String samplePath
    test('to test the property `samplePath`', () async {
      // TODO
    });

    // The built-in preset templates, in the settings section's dropdown order: date nestings, album layouts, camera groupings, the type grouping, the asset-id fallback and the bare file name. 
    // BuiltList<String> presets
    test('to test the property `presets`', () async {
      // TODO
    });

  });
}
