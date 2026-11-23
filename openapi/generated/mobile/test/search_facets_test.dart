import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for SearchFacets
void main() {
  final instance = SearchFacetsBuilder();
  // TODO add properties to the builder and call build()

  group(SearchFacets, () {
    // The caller's named people: one entry per labeled face cluster with a face on the current set. 
    // BuiltList<PersonFacetCount> people
    test('to test the property `people`', () async {
      // TODO
    });

    // The resolved places of the current set with the locale display name (`locale` parameter, the `GET /places` join). 
    // BuiltList<PlaceFacetCount> places
    test('to test the property `places`', () async {
      // TODO
    });

    // The taken-at years of the current set, newest first: the time-range facet (the `from`/`to` filters are the dimension the year counts exclude). 
    // BuiltList<YearFacetCount> years
    test('to test the property `years`', () async {
      // TODO
    });

    // The EXIF camera makes/models of the current set. The `id` is the round-trip filter value (`make|model`); `make` and `model` are the display halves, null exactly when the EXIF row carries no value there. 
    // BuiltList<CameraFacetCount> cameras
    test('to test the property `cameras`', () async {
      // TODO
    });

    // The media families of the current set.
    // BuiltList<KindFacetCount> kinds
    test('to test the property `kinds`', () async {
      // TODO
    });

    // The caller-visible albums (own and shared, never the sync albums -- the timeline is their face) holding members of the current set. 
    // BuiltList<AlbumFacetCount> albums
    test('to test the property `albums`', () async {
      // TODO
    });

    // The CALLER's tags with members in the current set (another account's tags never surface). 
    // BuiltList<TagFacetCount> tags
    test('to test the property `tags`', () async {
      // TODO
    });

  });
}
