import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for AlbumSummary
void main() {
  //final instance = AlbumSummaryBuilder();
  // TODO add properties to the builder and call build()

  group(AlbumSummary, () {
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

    // The album's type. `virtual` albums are curated by their owner; `directory` albums mirror a filesystem path. The one `sync` card, when present, is the caller's own default album pinned first (never another account's): its `title` is the internal system label ('Sync') and clients render the localized display name instead (en \"My photos\", zh-Hans \"我的照片\"). 
    // String type
    test('to test the property `type`', () async {
      // TODO
    });

    // The owner's lock: a locked album refuses its owner's curation and property writes until unlocked. It is metadata to a viewer (viewers are read-only regardless). 
    // bool locked
    test('to test the property `locked`', () async {
      // TODO
    });

    // True when the album carries at least one `album_shares` row -- someone besides the owner can see it. Always true on shared-group items by construction. 
    // bool isShared
    test('to test the property `isShared`', () async {
      // TODO
    });

    // The collage source of the wall card: the pinned cover asset as a single-element list when the owner set one, otherwise the album's most recent live member assets (up to four, `takenAt` descending, the asset id as the tiebreak). Empty for an album with no live members. 
    // BuiltList<String> coverAssetIds
    test('to test the property `coverAssetIds`', () async {
      // TODO
    });

    // The album's live member count: trashed, hidden and Live Photo component rows never count (a hidden asset is invisible to the whole family). 
    // int itemCount
    test('to test the property `itemCount`', () async {
      // TODO
    });

    // When the album row last changed (RFC 3339 UTC).
    // DateTime updatedAt
    test('to test the property `updatedAt`', () async {
      // TODO
    });

    // The caller's relation to this album: `owner` items belong to the own group, `viewer` items to the shared group. The web client sections the wall off this discriminator. 
    // String callerRole
    test('to test the property `callerRole`', () async {
      // TODO
    });

    // The owner attribution of a shared item: the account that owns the album. Present exactly when `callerRole` is `viewer`; own items omit it (the owner is the caller). 
    // AlbumOwnerRef owner
    test('to test the property `owner`', () async {
      // TODO
    });

    // The caller's timeline preference of this album: whether it feeds their aggregated timeline. Present exactly on the excludable cards -- the own `directory` items and every shared item; the own `virtual` items omit it (they never feed the caller's own timeline, so there is nothing to exclude), and the pinned `sync` card omits it too (the core stream is never excludable). The value is the preference row's switch, true when no row exists (the default). 
    // bool inTimeline
    test('to test the property `inTimeline`', () async {
      // TODO
    });

  });
}
