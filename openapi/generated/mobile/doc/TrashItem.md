# lumemor_api.model.TrashItem

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**id** | **String** | UUIDv7 identifier of the asset. | 
**ownerId** | **String** | UUIDv7 identifier of the owning account. | 
**kind** | **String** | Media family of the asset. | 
**originalFilename** | **String** | The file name the asset was uploaded with. | 
**fileSize** | **int** | Size of the stored file in bytes. | 
**mimeType** | **String** | Canonical content type of the stored format, for example `image/jpeg` or `video/quicktime`.  | 
**checksum** | **String** | SHA-256 digest of the file bytes, lowercase hex. The prefix of the entity tags the media endpoints serve.  | 
**createdAt** | [**DateTime**](DateTime.md) | When the asset record was created (RFC 3339 UTC). | 
**takenAt** | [**DateTime**](DateTime.md) | Capture time (RFC 3339), the value the timeline would order the asset by.  | 
**takenAtSource** | **String** | Where `takenAt` comes from. | 
**isReady** | **bool** | True once the whole pipeline of the asset (metadata and serving copies) completed.  | 
**width** | **int** | Pixel width in sensor order; null until the metadata job refined the dimensions.  | 
**height** | **int** | Pixel height in sensor order; null until the metadata job refined the dimensions.  | 
**durationMs** | **int** | Playback duration of a video in milliseconds; null for images and until the metadata job refined it.  | 
**blurhash** | **String** | The blurhash of the tile for placeholder rendering; null until the thumbnail job stored it.  | 
**livePhotoVideoId** | **String** | For a Live Photo image, the id of its linked component video row (restored and expired together with the photo); null otherwise.  | 
**deletedAt** | [**DateTime**](DateTime.md) | When the asset was soft deleted (RFC 3339 UTC). The retention window ends at `deletedAt` plus the instance's `trash_retention_days`.  | 
**remainingDays** | **int** | Whole days until the expiry job removes the asset, rounded up (an asset trashed moments ago with the default retention of 30 days shows 30). 0 once the deadline has passed -- the row survives only until the next hourly expiry run.  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


