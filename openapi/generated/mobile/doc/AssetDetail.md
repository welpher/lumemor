# lumemor_api.model.AssetDetail

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**id** | **String** | UUIDv7 identifier of the asset. | 
**ownerId** | **String** | UUIDv7 identifier of the owning account. | 
**kind** | **String** | Media family of the asset. A Live Photo is an `image` whose video component is stored as a linked `video` row that never appears in the timeline.  | 
**originalFilename** | **String** | The file name the asset was uploaded with; the original download re-uses it (RFC 5987 escaped) for `Content-Disposition`.  | 
**fileSize** | **int** | Size of the stored file in bytes. | 
**mimeType** | **String** | Canonical content type of the stored format, for example `image/jpeg` or `video/quicktime`.  | 
**checksum** | **String** | SHA-256 digest of the file bytes, lowercase hex. The prefix of the entity tags the media endpoints serve.  | 
**createdAt** | [**DateTime**](DateTime.md) | When the asset record was created (RFC 3339 UTC). | 
**takenAt** | [**DateTime**](DateTime.md) | Capture time (RFC 3339). Seeded from `deviceCreatedAt` (or the upload time when absent) and refined from EXIF by the metadata pipeline; the response renders the best value the server has at answer time.  | 
**takenAtSource** | **String** | Where `takenAt` comes from: `import` is the device-reported capture time of the upload, `exif` an EXIF capture time the metadata job resolved with an offset, `file` a file timestamp (reserved for the future directory-album importer).  | 
**isTrashed** | **bool** | True when the asset sits in the trash. | 
**isHidden** | **bool** | True while the owner-level visibility switch is off: the asset is out of every live view (the timeline, the favorites) and manageable only through the hidden view, until it is unhidden. The same switch `PATCH /assets/{id}` and the batch `hide` / `unhide` actions flip.  | 
**description** | **String** | The owner's free-text description of the asset, the text the detail panel renders; the empty string until the owner wrote one. Editable only by the owner through `PATCH /assets/{id}`, which replaces it wholesale.  | 
**isReady** | **bool** | True once the whole pipeline of the asset completed -- the metadata and thumbnail jobs for an image, plus the transcode job for a video (for a Live Photo component, the transcode job alone); the `asset.ready` event fires at that moment. Until then the client shows placeholders and the media endpoints answer `ASSET_NOT_READY` for missing derived files.  | 
**width** | **int** | Pixel width in sensor order (not orientation-corrected); null until the metadata job refined the dimensions.  | 
**height** | **int** | Pixel height in sensor order (not orientation-corrected); null until the metadata job refined the dimensions.  | 
**durationMs** | **int** | Playback duration of a video in milliseconds; null for images and until the metadata job refined it.  | 
**blurhash** | **String** | The blurhash of the tile (image assets) or poster frame (video assets) for placeholder rendering; null until the thumbnail job stored it.  | 
**livePhotoVideoId** | **String** | For a Live Photo image, the id of its linked component video row (never in the timeline; its original is downloadable by that id, and its transcoded play stream is served there too); null otherwise.  | 
**exif** | [**ExifSummary**](ExifSummary.md) | The EXIF hot fields. Always present as an object; every field inside it is null until the metadata job wrote the extraction.  | 
**faces** | [**BuiltList&lt;AssetFaceSummary&gt;**](AssetFaceSummary.md) | The asset's faces as the calling user sees them, strongest score first with the face id as the tiebreak. The array is caller-scoped (B4): the asset being visible to the caller is the gate -- the faces ride on the detail's own permission -- and each entry's `label` is the caller's own label of the face's cluster, never another account's. Always present, empty when the asset carries no faces (none detected, or detection has not run yet). The PATCH response serves the same array, exactly as the GET serves it.  | 
**tags** | [**BuiltList&lt;AssetTagSummary&gt;**](AssetTagSummary.md) | The CALLER's tags on this asset (B4): the caller's own assignment rows joined to their own tag rows, sorted by name case-insensitively with the tag id as the tiebreak (the vocabulary listing's own ordering). Another account's tags on the same asset never surface, so the same shared asset answers different arrays for the owner and the sharee (each sees their own, or none). Always present, empty when the caller tagged the asset with nothing. The PATCH response serves the same array, exactly as the GET serves it.  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


