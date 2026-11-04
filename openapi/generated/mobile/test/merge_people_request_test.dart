import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for MergePeopleRequest
void main() {
  final instance = MergePeopleRequestBuilder();
  // TODO add properties to the builder and call build()

  group(MergePeopleRequest, () {
    // The clusters to fold into the target: at least 1 and at most 50 after duplicates collapse, every one visible to the caller. Each dissolves with its label rows. A single id is the pairwise fold of the cluster detail's \"merge into\" action. 
    // BuiltList<String> sourceClusterIds
    test('to test the property `sourceClusterIds`', () async {
      // TODO
    });

    // The surviving cluster: every source face re-homes here and its representative recomputes to the strongest member of the merged set. It must be visible to the caller and must not sit among the sources. 
    // String targetClusterId
    test('to test the property `targetClusterId`', () async {
      // TODO
    });

  });
}
