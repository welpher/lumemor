import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for AlbumWallResponse
void main() {
  final instance = AlbumWallResponseBuilder();
  // TODO add properties to the builder and call build()

  group(AlbumWallResponse, () {
    // Every wall album: the pinned sync card first (the caller's own core upload stream, the default \"My photos\" album), then the own group (the caller's virtual and directory albums, `updatedAt` descending), then the shared group (the albums shared to the caller, the caller's grant time descending). Each group breaks ties on the album id, descending. A sync album another account owns never appears. 
    // BuiltList<AlbumSummary> albums
    test('to test the property `albums`', () async {
      // TODO
    });

  });
}
