# lumemor_api.model.AlbumPatch

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**title** | **String** | The album's display title, trimmed server-side; when present it must span 1-64 Unicode characters after the trim.  | [optional] 
**coverAssetIds** | **BuiltList&lt;String&gt;** | The owner's pin, as a full set: the frames of the cover in render order (the order the owner picked them in -- the server stores and returns it as given, never re-sorting). The length is 0 or exactly N, N being the card's frame count (four on the album card, six on the memory band): an empty array clears the pin and returns the album to the automatic weekly group, a full set pins it, and any other length answers 400 `VALIDATION_FAILED` (a partial set is the picker's un-saveable state, never a stored one). Repeated ids answer 400 too -- a cover is a set, and one frame cannot fill two cells. Every id must be a RENDERABLE member of THIS album (the membership row exists and the asset is not trashed, is not a Live Photo component row, and on a memory album is not archived -- an album's archived members render and may pin); otherwise the answer is 400 with the message key `album_cover_not_member` and a `count` of the refused frames. Absent (or JSON null, which decodes as absent) leaves the pin unchanged.  | [optional] 
**allowAppWrites** | **bool** | The owner's write switch of a DIRECTORY album: whether the application may modify the album's source files -- delete them through the trash move, and write a capture-time edit back into their metadata. On for a writable source by default; the owner who manages the files themselves turns it off, and the wall's delete affordances and the assets' time edits disappear with it (the derived `deletable` and `metadataWritable` faces). A patch carrying the field on a memory album answers 400 with the message key `album_deletable_not_directory` -- the switch is the directory album's own, a memory album's deletions never move files. Idempotent: repeating the stored value converges.  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


