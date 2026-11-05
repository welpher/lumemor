# lumemor_api.model.TimelineItem

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**id** | **String** | UUIDv7 identifier of the asset. | 
**ownerId** | **String** | UUIDv7 identifier of the owning account. | 
**kind** | **String** | Media family of the asset. A Live Photo is an `image` whose linked component video row never appears in the timeline.  | 
**originalFilename** | **String** | The file name the asset was uploaded with. | 
**fileSize** | **int** | Size of the stored file in bytes. | 
**mimeType** | **String** | Canonical content type of the stored format, for example `image/jpeg` or `video/quicktime`.  | 
**checksum** | **String** | SHA-256 digest of the file bytes, lowercase hex. The prefix of the entity tags the media endpoints serve.  | 
**createdAt** | [**DateTime**](DateTime.md) | When the asset record was created (RFC 3339 UTC). | 
**takenAt** | [**DateTime**](DateTime.md) | Capture time (RFC 3339). Seeded from `deviceCreatedAt` (or the upload time when absent) and refined from EXIF by the metadata pipeline; the timeline orders by it.  | 
**takenAtSource** | **String** | Where `takenAt` comes from: `import` is the device-reported capture time of the upload, `exif` an EXIF capture time the metadata job resolved with an offset, `file` a file timestamp (reserved for the future directory-album importer).  | 
**isTrashed** | **bool** | Always false in the timeline: trashed assets never enter it. The field exists to keep the shape identical to the asset detail's.  | 
**isFavorite** | **bool** | Whether the REQUESTING user favorites this asset: the caller's own row of the per-user favorites state, written by the batch endpoint's favorite action. False whenever the caller has no favorites row for the asset -- the same asset can be a favorite for one viewer of a shared timeline and not for another.  | 
**isReady** | **bool** | True once the whole pipeline of the asset (metadata and serving copies) completed; until then the pipeline-dependent fields are null and the client renders placeholders.  | 
**width** | **int** | Pixel width in sensor order, straight from the assets column: the upload writes the 0 placeholder and the metadata job refines the real value, so 0 means \"not refined yet\" (the wall layout assumes a fallback aspect until then). Always present -- the justified photo wall reads it on every tile, never null.  | 
**height** | **int** | Pixel height in sensor order, straight from the assets column: the upload writes the 0 placeholder and the metadata job refines the real value, so 0 means \"not refined yet\" (the wall layout assumes a fallback aspect until then). Always present -- the justified photo wall reads it on every tile, never null.  | 
**durationMs** | **int** | Playback duration of a video in milliseconds; null for images and until the metadata job refined it.  | 
**blurhash** | **String** | The blurhash of the tile (image assets) or poster frame (video assets) for placeholder rendering; null until the thumbnail job stored it.  | 
**livePhotoVideoId** | **String** | For a Live Photo image, the id of its linked component video row (its original is downloadable by that id, the transcoded play stream is not); null otherwise.  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


