# lumemor_api.model.AssetAlbumRef

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**id** | **String** | Identifier of the album (the detail face's deep link). | 
**title** | **String** | The album's display title (the sync card's internal label). | 
**type** | **String** | The album's type. A `memory` entry is one of the viewer's own or shared reference sets; `directory` entries are sources. The one `sync` entry, when present, is the viewer's own default album (never another account's).  | 
**callerRole** | **String** | The caller's relation to this album: `owner` items are the caller's own (the sync stream's entry is always one), `viewer` items the albums shared to them.  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


