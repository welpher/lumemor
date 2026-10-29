import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for Error
void main() {
  final instance = ErrorBuilder();
  // TODO add properties to the builder and call build()

  group(Error, () {
    // Stable error code. The i18n translation key is the lowercase form of this value. 
    // String code
    test('to test the property `code`', () async {
      // TODO
    });

    // English technical description for debugging.
    // String message
    test('to test the property `message`', () async {
      // TODO
    });

    // Optional structured context, for example the offending field names of a `VALIDATION_FAILED` request. Its shape is code-specific. 
    // JsonObject details
    test('to test the property `details`', () async {
      // TODO
    });

  });
}
