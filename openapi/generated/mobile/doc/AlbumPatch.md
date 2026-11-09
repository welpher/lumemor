# lumemor_api.model.AlbumPatch

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**title** | **String** | The album's display title, trimmed server-side; when present it must span 1-64 Unicode characters after the trim.  | [optional] 
**coverAssetId** | **String** | The pinned cover asset. When present it must be a LIVE member of THIS album (the membership row exists and the asset is not trashed, not hidden, not a Live Photo component row) -- otherwise the answer is 400 with the message key `album_cover_not_member`. Absent (or JSON null, which decodes as absent) leaves the pin unchanged.  | [optional] 
**locked** | **bool** | The owner's lock, the content-protection switch: `true` freezes the album's curation and property fields against later writes (this same endpoint refuses them with 409 until the owner patches `locked: false` -- the unlock, never gated itself) AND freezes the content itself -- the directory rescan and the deletion of the album's photos answer the same 409 while the lock holds. The audience face stays open under a lock (its own pin: a lock protects content, not visibility), and the album's own delete answers the same 409 every write does.  | [optional] 
**allowAppDeletion** | **bool** | The owner's deletability switch of a DIRECTORY album: whether the album's photos may be deleted through the application (the trash move of the source file). On for a writable source by default; the owner who manages file-system deletions themselves turns it off, and the wall's delete affordances disappear with it (the derived `deletable` face of the detail read). A patch carrying the field on a virtual album answers 400 with the message key `album_deletable_not_directory` -- the switch is the directory album's own, a virtual album's deletions never move files. Idempotent: repeating the stored value converges, and the switch joins the property fields the lock refuses (409 until unlocked).  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


