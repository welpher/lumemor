import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for UnnamedClustersResponse
void main() {
  final instance = UnnamedClustersResponseBuilder();
  // TODO add properties to the builder and call build()

  group(UnnamedClustersResponse, () {
    // The page's clusters, ordered by `faceCount` descending with the cluster id as the tiebreak. 
    // BuiltList<UnnamedClusterSummary> items
    test('to test the property `items`', () async {
      // TODO
    });

    // The opaque continuation cursor: pass it back as the `cursor` query parameter to fetch the next page. Absent exactly when this is the last page. 
    // String nextCursor
    test('to test the property `nextCursor`', () async {
      // TODO
    });

    // The caller's whole unnamed region: every cluster this listing would walk, independent of the page size and the cursor. The people list's unnamed entry badge and the unnamed page's own count both read it. 
    // int totalCount
    test('to test the property `totalCount`', () async {
      // TODO
    });

  });
}
