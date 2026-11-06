import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for SearchResponse
void main() {
  final instance = SearchResponseBuilder();
  // TODO add properties to the builder and call build()

  group(SearchResponse, () {
    // The page's assets, ordered by the fused score descending with the asset id ascending as the tiebreak (a text query), or by taken-at descending with the id descending (the filter browse -- the timeline's own order). 
    // BuiltList<TimelineItem> items
    test('to test the property `items`', () async {
      // TODO
    });

    // The continuation token of the page: present exactly when another page follows. Opaque; pass it back as `cursor`. 
    // String nextCursor
    test('to test the property `nextCursor`', () async {
      // TODO
    });

    // SearchFacets facets
    test('to test the property `facets`', () async {
      // TODO
    });

  });
}
