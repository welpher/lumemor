import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for PersonLabel
void main() {
  final instance = PersonLabelBuilder();
  // TODO add properties to the builder and call build()

  group(PersonLabel, () {
    // The caller's name for the person.
    // String name
    test('to test the property `name`', () async {
      // TODO
    });

    // The caller's hidden switch. Hidden people stay listed in `GET /people` carrying this flag (badge semantics). 
    // bool isHidden
    test('to test the property `isHidden`', () async {
      // TODO
    });

  });
}
