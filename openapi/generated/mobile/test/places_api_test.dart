import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';


/// tests for PlacesApi
void main() {
  final instance = LumemorApi().getPlacesApi();

  group(PlacesApi, () {
    // The map's client-side tile configuration.
    //
    // The configuration the places map and the viewer's location card need before they can draw a base layer: the tile source the administrator configured, and nothing else.  THE POLYGON: the value is a URL the BROWSER loads directly from the tile host -- the server never fetches, proxies or caches a tile, so a pan or a zoom never makes the photo server a tile CDN. What the tile host sees is the viewport, the zoom level and the browsing device's address; the pins the map draws come from this instance's own API and never cross the tile line. A deployment whose browser cannot reach a public tile host points the setting at a self-hosted or mirrored source; there is deliberately no bundled default (a public tile service's usage policy forbids carrying the traffic of an application's whole install base).  `tilejsonUrl` is null when no source is configured. That is not an error: the map renders its honest degraded state -- the pins stay, the base layer is stated as absent -- and the admin settings face is where it gets fixed. A stored value is handed back verbatim; its reachability is the browser's business, and a source that fails to load lands on the same stated absence. 
    //
    //Future<MapConfigResponse> getMapConfig() async
    test('test getMapConfig', () async {
      // TODO
    });

    // The caller's city grouping.
    //
    // The places surface: every place of the GeoNames reference data that holds at least one asset the caller can see -- the city grouping the places page renders as its grid. A place joins the list when it carries a caller-visible live asset (the album union of the caller's own sync and directory albums plus every album shared to them, and the timeline's live-row filters: not trashed, not hidden, no Live Photo component row); `assetCount` counts exactly those assets, so the same place can show different counts to different accounts -- the honest per-caller counting discipline the people surface pins, applied to cities.  `name` is the locale-joined display name: the `place_names` row of the requested locale when the place data carries one, the GeoNames canonical (English or romanized) name otherwise. Locale is a REQUEST parameter (the web app owns the active locale and passes it along); `en` is the default. A locale switch changes the join only -- the assets rows never carry locale strings, so switching languages never rewrites asset data.  `representativeCoverAssetIds` is the wall's derived-cover discipline applied to city groups: the place's (up to) four most recent caller-visible live members, taken_at descending with the asset id as the tiebreak -- the cover tiles the places page renders from. A listed place always carries at least one candidate (the listing itself proved a visible member exists).  `takenAtFrom` and `takenAtTo` are the place's caller-visible live members' earliest and latest capture times: the span the city card prints as its year range (\"2019-2026\", one year when the two fall in the same calendar year). A listed place always carries both.  The list is ordered by `takenAtTo` descending -- the most recently photographed city first -- with the place id ascending as the deterministic tiebreak. It is not paginated: the same single-document shape the named-people listing uses, sized for a family instance's few thousand places.  Place resolution itself (which asset belongs to which city) is the ingestion pipeline's business -- the metadata job's GPS step and the admin re-resolution run -- and follows the place data baked into the server image; this endpoint only reads what those wrote. 
    //
    //Future<BuiltList<PlaceSummary>> listPlaces({ String locale }) async
    test('test listPlaces', () async {
      // TODO
    });

  });
}
