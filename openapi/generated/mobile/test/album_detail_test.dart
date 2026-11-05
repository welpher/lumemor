import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for AlbumDetail
void main() {
  final instance = AlbumDetailBuilder();
  // TODO add properties to the builder and call build()

  group(AlbumDetail, () {
    // Identifier of the album.
    // String id
    test('to test the property `id`', () async {
      // TODO
    });

    // The album's display title.
    // String title
    test('to test the property `title`', () async {
      // TODO
    });

    // The album's free-text description (empty when unset).
    // String description
    test('to test the property `description`', () async {
      // TODO
    });

    // The album's type; sync albums never reach this shape.
    // String type
    test('to test the property `type`', () async {
      // TODO
    });

    // The owner's lock (metadata to a viewer).
    // bool locked
    test('to test the property `locked`', () async {
      // TODO
    });

    // The pinned cover asset, or null when unset -- the raw column; the wall derives its collage fallback, the detail carries the pin itself. 
    // String coverAssetId
    test('to test the property `coverAssetId`', () async {
      // TODO
    });

    // The caller's relation to this album: the owner holds the write affordances, a viewer is read-only. 
    // String callerRole
    test('to test the property `callerRole`', () async {
      // TODO
    });

    // The album's live member count, the same predicate as the items below. 
    // int itemCount
    test('to test the property `itemCount`', () async {
      // TODO
    });

    // One page of the album's live members, `takenAt` descending with the asset id as the tiebreak. 
    // BuiltList<AlbumItem> items
    test('to test the property `items`', () async {
      // TODO
    });

    // The opaque continuation cursor: pass it back as the `cursor` query parameter to fetch the next page. Absent exactly when this is the last page. 
    // String nextCursor
    test('to test the property `nextCursor`', () async {
      // TODO
    });

    // The album's grantee list, riding ONLY the owner's view: one entry per `album_shares` row in the grant order (`grantedAt` ascending) -- the share management dialog's list. The owner's response always carries the field, an empty array when the album holds no grants; viewer responses never carry it at all. 
    // BuiltList<AlbumShareEntry> shares
    test('to test the property `shares`', () async {
      // TODO
    });

    // The caller's timeline preference of this album: whether it feeds their aggregated timeline. Present exactly when the album is excludable for the caller -- an own `directory` album or an album shared to them; an own `virtual` album omits the field (it never feeds the caller's own timeline). The value is the preference row's switch, true when no row exists (the default). The preference scopes the aggregated timeline only: it never filters this album's own items, whatever the switch says. 
    // bool inTimeline
    test('to test the property `inTimeline`', () async {
      // TODO
    });

  });
}
