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

    // The album's type. `memory` albums are reference sets curated by their owner -- photos live in the albums they were ingested through, a memory holds references to them; `directory` albums mirror a filesystem path. The one `sync` card, when present, is the caller's own default album pinned first (never another account's): its `title` is the internal system label ('Sync') and clients render the localized display name instead (en \"My photos\", zh-Hans \"我的照片\"). 
    // String type
    test('to test the property `type`', () async {
      // TODO
    });

    // True when the album carries at least one `album_shares` row -- someone besides the owner can see it. Always true on shared-group items by construction. 
    // bool isShared
    test('to test the property `isShared`', () async {
      // TODO
    });

    // The collage source of the wall card, in render order: the frames of the album's cover. A cover holds four frames (the album card's 2x2 collage) or six (the memory band's 3-across 2-down strip), and the server answers the group the card draws -- the card is the first thing rendered and cannot see the collection, so the automatic group is computed here, never by the client.  The owner's pinned set wins: when the owner picked a full set, these are its frames in pick order, exactly as written (the server never re-sorts a pin). An album without pins renders the automatic group: a weekly rotation, re-sampled per ISO week from the whole live collection (never the newest frames and never a random order), frozen for the week so an upload mid-week cannot move a frame, and never rotating at all while the album holds fewer than twice the frame count -- such an album keeps one stable sample of its own. A pinned frame that is no longer visible to the caller (moved out, trashed, hidden) is filled silently from the caller's visible pool; a fully dead pin falls back to the automatic group.  The order is the render order and the length is at most the card's frame count: a collection with fewer live members than frames answers the frames it has (one fills, two split, three lay one large beside two small), and an album with no live members answers an empty array (the card's empty cover, no placeholder photo). 
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

    // The caller's relation to this album: `owner` items belong to the caller's own group on the wire, `viewer` items to the shared group. The client renders ONE wall list and keys each card's face off this discriminator (the read-only mark is the member's, the shared state the owner's). 
    // String callerRole
    test('to test the property `callerRole`', () async {
      // TODO
    });

    // The owner attribution of a shared item: the account that owns the album. Present exactly when `callerRole` is `viewer`; own items omit it (the owner is the caller). 
    // AlbumOwnerRef owner
    test('to test the property `owner`', () async {
      // TODO
    });

    // The caller's timeline preference of this album: whether it feeds their aggregated timeline. Present exactly on the excludable cards -- the own `directory` items and every shared item; the own `memory` items omit it (they never feed the caller's own timeline, so there is nothing to exclude), and the pinned `sync` card omits it too (the core stream is never excludable). The value is the preference row's switch, true when no row exists (the default). 
    // bool inTimeline
    test('to test the property `inTimeline`', () async {
      // TODO
    });

  });
}
