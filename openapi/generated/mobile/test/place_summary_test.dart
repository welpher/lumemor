import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for PlaceSummary
void main() {
  final instance = PlaceSummaryBuilder();
  // TODO add properties to the builder and call build()

  group(PlaceSummary, () {
    // The GeoNames geonameid of the place -- the `places` row's primary key and the `assets.place_id` foreign key the ingestion pipeline's resolution writes. 
    // int placeId
    test('to test the property `placeId`', () async {
      // TODO
    });

    // The display name in the request's locale: the `place_names` row of that locale when the place data carries one, the GeoNames canonical (English or romanized) name otherwise. A locale switch changes this join only, never an asset row. 
    // String name
    test('to test the property `name`', () async {
      // TODO
    });

    // The place's ISO 3166-1 alpha-2 country code. The display country name is the client's business (its own locale table maps the code). 
    // String country
    test('to test the property `country`', () async {
      // TODO
    });

    // The place's caller-visible live assets -- the honest per-caller count, not the global member count. 
    // int assetCount
    test('to test the property `assetCount`', () async {
      // TODO
    });

    // The place's cover candidates: its (up to) four most recent caller-visible live members, taken_at descending with the asset id as the tiebreak -- the wall's derived-cover discipline applied to city groups, newest first. 
    // BuiltList<String> representativeCoverAssetIds
    test('to test the property `representativeCoverAssetIds`', () async {
      // TODO
    });

  });
}
