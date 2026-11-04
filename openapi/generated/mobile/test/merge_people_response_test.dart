import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for MergePeopleResponse
void main() {
  final instance = MergePeopleResponseBuilder();
  // TODO add properties to the builder and call build()

  group(MergePeopleResponse, () {
    // The target cluster that absorbed the sources.
    // String survivingClusterId
    test('to test the property `survivingClusterId`', () async {
      // TODO
    });

    // The source clusters that dissolved into the target, in request order. Their rows -- and every label on them -- are gone. 
    // BuiltList<String> mergedClusterIds
    test('to test the property `mergedClusterIds`', () async {
      // TODO
    });

  });
}
