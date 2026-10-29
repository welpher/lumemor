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
**isReady** | **bool** | True once the whole pipeline of the asset (metadata and serving copies) completed; the `asset.ready` event fires at that moment. Until then the client shows placeholders and the media endpoints answer `ASSET_NOT_READY` for missing derived files.  | 
**width** | **int** | Pixel width in sensor order (not orientation-corrected); null until the metadata job refined the dimensions.  | 
**height** | **int** | Pixel height in sensor order (not orientation-corrected); null until the metadata job refined the dimensions.  | 
**durationMs** | **int** | Playback duration of a video in milliseconds; null for images and until the metadata job refined it.  | 
**blurhash** | **String** | The blurhash of the tile (image assets) or poster frame (video assets) for placeholder rendering; null until the thumbnail job stored it.  | 
**livePhotoVideoId** | **String** | For a Live Photo image, the id of its linked component video row (never in the timeline, but resolvable for playback); null otherwise.  | 
**exif** | [**ExifSummary**](ExifSummary.md) | The EXIF hot fields. Always present as an object; every field inside it is null until the metadata job wrote the extraction.  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


