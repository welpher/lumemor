import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for AssetPlaceRef
void main() {
  final instance = AssetPlaceRefBuilder();
  // TODO add properties to the builder and call build()

  group(AssetPlaceRef, () {
    // The GeoNames geonameid of the place -- the `places` row's primary key and the `assets.place_id` foreign key, the same identifier the place-filtered browse filters on. 
    // int placeId
    test('to test the property `placeId`', () async {
      // TODO
    });

    // The display name at the request's locale: the `place_names` row of that locale when the place data carries one, the GeoNames canonical (English or romanized) name otherwise. 
    // String name
    test('to test the property `name`', () async {
      // TODO
    });

  });
}
