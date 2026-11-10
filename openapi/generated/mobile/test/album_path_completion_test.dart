import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for AlbumPathCompletion
void main() {
  final instance = AlbumPathCompletionBuilder();
  // TODO add properties to the builder and call build()

  group(AlbumPathCompletion, () {
    // Full absolute directory path of the container's visible filesystem, never inside the server's managed tree. 
    // String path
    test('to test the property `path`', () async {
      // TODO
    });

    // The directory's write fact as the listing discovered it: true when the probe file landed, read back and removed cleanly; false when any step refused. The fact is information, never an error -- a false row lists alongside the true ones, and picking it creates a legitimate read-only-source album. 
    // bool writable
    test('to test the property `writable`', () async {
      // TODO
    });

  });
}
