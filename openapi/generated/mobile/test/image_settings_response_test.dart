import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for ImageSettingsResponse
void main() {
  final instance = ImageSettingsResponseBuilder();
  // TODO add properties to the builder and call build()

  group(ImageSettingsResponse, () {
    // The timeline tile's bounding box in pixels: 250 is the shipped default, 320 reads sharper on high-density screens. 
    // int tileSize
    test('to test the property `tileSize`', () async {
      // TODO
    });

    // The viewer's inline preview long edge in pixels: 1080 is the shipped default (enough to look at, cheapest to derive), 1440 and 2048 zoom sharper at a bigger encode. 
    // int previewSize
    test('to test the property `previewSize`', () async {
      // TODO
    });

    // DerivativeFormat format
    test('to test the property `format`', () async {
      // TODO
    });

    // Whether derivations also produce the full-resolution serving copy. Shipped OFF: the viewer reads the preview, and the full-size copy is disk the family never opens. 
    // bool fullSize
    test('to test the property `fullSize`', () async {
      // TODO
    });

    // Whether a RAW source derives from its embedded preview. Shipped ON. 
    // bool rawEmbedded
    test('to test the property `rawEmbedded`', () async {
      // TODO
    });

  });
}
