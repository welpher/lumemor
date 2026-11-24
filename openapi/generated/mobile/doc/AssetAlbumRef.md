# lumemor_api.model.AssetAlbumRef

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**id** | **String** | Identifier of the album (the detail face's deep link). | 
**allowAppWrites** | **bool** | The album's write switch as its OWNER set it. True (the creation default) means the application may modify the album's source files -- delete them through the trash move and write capture-time edits into their metadata. Carried on the owner's rows; a viewer's row carries false (a viewer holds no write to read it for). The detail's delete affordance folds this with `sourceWritable`.  | 
**sourceWritable** | **bool** | Whether the album's source directory can take the application's writes. False only for a directory album whose creation-time write probe answered no -- every other type (sync, memory, a writable directory) is true. The photo-detail faces read this to gate the destructive affordances: a photo housed only in a read-only source is the scan's verdict alone and offers no delete.  | 
**title** | **String** | The album's display title (the sync card's internal label). | 
**type** | **String** | The album's type. A `memory` entry is one of the viewer's own or shared reference sets; `directory` entries are sources. The one `sync` entry, when present, is the viewer's own default album (never another account's).  | 
**callerRole** | **String** | The caller's relation to this album: `owner` items are the caller's own (the sync stream's entry is always one), `viewer` items the albums shared to them.  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


