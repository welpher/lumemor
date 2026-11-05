# lumemor_api.model.AlbumPatch

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**title** | **String** | The album's display title, trimmed server-side; when present it must span 1-64 Unicode characters after the trim.  | [optional] 
**description** | **String** | The album's free-text description; the value replaces the stored text wholesale (an empty string clears it). At most 10,000 Unicode characters.  | [optional] 
**coverAssetId** | **String** | The pinned cover asset. When present it must be a LIVE member of THIS album (the membership row exists and the asset is not trashed, not hidden, not a Live Photo component row) -- otherwise the answer is 400 with the message key `album_cover_not_member`. Absent (or JSON null, which decodes as absent) leaves the pin unchanged.  | [optional] 
**locked** | **bool** | The owner's lock. `true` freezes the album's curation and property fields against later writes (this same endpoint refuses them with 409 until the owner patches `locked: false` -- the unlock, never gated itself). Locking does not gate the album's delete.  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


