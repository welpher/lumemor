import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for AssetPatch
void main() {
  final instance = AssetPatchBuilder();
  // TODO add properties to the builder and call build()

  group(AssetPatch, () {
    // The owner's free-text description of the asset, the text the detail panel renders. The value replaces the stored text wholesale (an empty string clears it). At most 10,000 Unicode characters -- the count is over runes, so a text of multi-byte characters stays within the bound at the same length. 
    // String description
    test('to test the property `description`', () async {
      // TODO
    });

    // The owner-level visibility switch: `true` keeps the asset out of every live view (the timeline, the favorites) until it is unhidden; `false` returns it. The same switch the batch `hide` / `unhide` actions flip. 
    // bool isHidden
    test('to test the property `isHidden`', () async {
      // TODO
    });

  });
}
