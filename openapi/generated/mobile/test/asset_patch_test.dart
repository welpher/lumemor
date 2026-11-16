import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for AssetPatch
void main() {
  final instance = AssetPatchBuilder();
  // TODO add properties to the builder and call build()

  group(AssetPatch, () {
    // The owner's free-text description of the asset, the text the detail panel renders. The value replaces the stored text wholesale (an empty string clears it). At most 10,000 Unicode characters -- the count is over runes, so a text of multi-byte characters stays within the bound at the same length. 
    // String description
    test('to test the property `description`', () async {
      // TODO
    });

    // The owner-level archive switch: `true` keeps the asset out of every family flow until it is unarchived; `false` returns it. An album keeps its archived members, so the album surface is unaffected either way. The same switch the batch `hide` / `unhide` actions flip. 
    // bool isArchived
    test('to test the property `isArchived`', () async {
      // TODO
    });

    // The owner's manual capture-time edit: the value replaces the stored instant wholesale. The timestamp MUST carry a UTC offset (RFC 3339's `Z` or `+hh:mm` designator) -- the offset is the capture zone the info panel displays beside the time, so a zone-less value is ambiguous and answers 400 `VALIDATION_FAILED`. The same value decides where the photo sits on the timeline (the instant) and which offset renders beside it (the offset). 
    // DateTime takenAt
    test('to test the property `takenAt`', () async {
      // TODO
    });

    // The manual location edit's latitude, in degrees (-90..90). It rides with `longitude` (the pair is the location; one half alone answers 400 `VALIDATION_FAILED`). Writing the pair replaces the asset's location: the info panel renders these coordinates and the resolved place follows the pin. The camera-recorded GPS in `exif` keeps the file's own reading and is not touched. 
    // double latitude
    test('to test the property `latitude`', () async {
      // TODO
    });

    // The manual location edit's longitude, in degrees (-180..180). It rides with `latitude` (see there for the pair's semantics). 
    // double longitude
    test('to test the property `longitude`', () async {
      // TODO
    });

  });
}
