import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for AlbumPathCompletionsResponse
void main() {
  final instance = AlbumPathCompletionsResponseBuilder();
  // TODO add properties to the builder and call build()

  group(AlbumPathCompletionsResponse, () {
    // The pickable directories of the container's visible filesystem -- outside the server's managed tree, real directories (symlinked entries are never listed). Deduplicated, sorted ascending by path, at most 20 rows. The empty array is an honest \"nothing matches\", never an error. Every row carries its write fact: the probe file was written, read back and removed cleanly (`writable` true), or any step refused (false -- a read-only directory is a legitimate source, and a false fact is information, never an error). 
    // BuiltList<AlbumPathCompletion> completions
    test('to test the property `completions`', () async {
      // TODO
    });

  });
}
