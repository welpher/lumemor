import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for AlbumWallResponse
void main() {
  final instance = AlbumWallResponseBuilder();
  // TODO add properties to the builder and call build()

  group(AlbumWallResponse, () {
    // Every wall album: the own group first (the caller's virtual and directory albums, `updatedAt` descending), then the shared group (the albums shared to the caller, the caller's grant time descending). Each group breaks ties on the album id, descending. Sync albums never appear in either group. 
    // BuiltList<AlbumSummary> albums
    test('to test the property `albums`', () async {
      // TODO
    });

  });
}
