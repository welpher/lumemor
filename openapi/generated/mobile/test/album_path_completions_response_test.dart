import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for AlbumPathCompletionsResponse
void main() {
  final instance = AlbumPathCompletionsResponseBuilder();
  // TODO add properties to the builder and call build()

  group(AlbumPathCompletionsResponse, () {
    // Full absolute directory paths of the container's visible filesystem -- outside the server's managed tree, real directories (symlinked entries are never listed). Deduplicated, sorted ascending, at most 20 rows. The empty array is an honest \"nothing matches\", never an error. 
    // BuiltList<String> paths
    test('to test the property `paths`', () async {
      // TODO
    });

  });
}
