import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for ExifSummary
void main() {
  final instance = ExifSummaryBuilder();
  // TODO add properties to the builder and call build()

  group(ExifSummary, () {
    // Camera manufacturer (EXIF Make).
    // String cameraMake
    test('to test the property `cameraMake`', () async {
      // TODO
    });

    // Camera model (EXIF Model).
    // String cameraModel
    test('to test the property `cameraModel`', () async {
      // TODO
    });

    // Lens identifier (LensModel, LensID or Lens).
    // String lens
    test('to test the property `lens`', () async {
      // TODO
    });

    // ISO sensitivity.
    // int iso
    test('to test the property `iso`', () async {
      // TODO
    });

    // Aperture as the f-number.
    // num aperture
    test('to test the property `aperture`', () async {
      // TODO
    });

    // Exposure time in photo notation, for example `1/250` or `2.5s`. 
    // String shutter
    test('to test the property `shutter`', () async {
      // TODO
    });

    // Focal length in millimeters.
    // num focal
    test('to test the property `focal`', () async {
      // TODO
    });

    // GPS latitude in degrees.
    // double lat
    test('to test the property `lat`', () async {
      // TODO
    });

    // GPS longitude in degrees.
    // double lon
    test('to test the property `lon`', () async {
      // TODO
    });

    // GPS altitude in meters.
    // num alt
    test('to test the property `alt`', () async {
      // TODO
    });

    // The offset that resolved the capture time, in EXIF notation (`+05:30`); null when the capture time did not resolve through EXIF. 
    // String tzOffset
    test('to test the property `tzOffset`', () async {
      // TODO
    });

  });
}
