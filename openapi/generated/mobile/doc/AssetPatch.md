# lumemor_api.model.AssetPatch

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**description** | **String** | The owner's free-text description of the asset, the text the detail panel renders. The value replaces the stored text wholesale (an empty string clears it). At most 10,000 Unicode characters -- the count is over runes, so a text of multi-byte characters stays within the bound at the same length.  | [optional] 
**isHidden** | **bool** | The owner-level visibility switch: `true` keeps the asset out of every live view (the timeline, the favorites) until it is unhidden; `false` returns it. The same switch the batch `hide` / `unhide` actions flip.  | [optional] 
**takenAt** | [**DateTime**](DateTime.md) | The owner's manual capture-time edit: the value replaces the stored instant wholesale. The timestamp MUST carry a UTC offset (RFC 3339's `Z` or `+hh:mm` designator) -- the offset is the capture zone the info panel displays beside the time, so a zone-less value is ambiguous and answers 400 `VALIDATION_FAILED`. The same value decides where the photo sits on the timeline (the instant) and which offset renders beside it (the offset).  | [optional] 
**latitude** | **double** | The manual location edit's latitude, in degrees (-90..90). It rides with `longitude` (the pair is the location; one half alone answers 400 `VALIDATION_FAILED`). Writing the pair replaces the asset's location: the info panel renders these coordinates and the resolved place follows the pin. The camera-recorded GPS in `exif` keeps the file's own reading and is not touched.  | [optional] 
**longitude** | **double** | The manual location edit's longitude, in degrees (-180..180). It rides with `latitude` (see there for the pair's semantics).  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


