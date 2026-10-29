# lumemor_api.model.AssetResponse

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
**originalFilename** | **String** | The file name the asset was uploaded with: the client-side `fileName` of the photo (the component video keeps its own file part name).  | 
**fileSize** | **int** | Size of the stored file in bytes. | 
**mimeType** | **String** | Canonical content type of the stored format, for example `image/jpeg` or `video/quicktime`.  | 
**checksum** | **String** | SHA-256 digest of the file bytes, lowercase hex. The owner-scoped idempotency key of the upload.  | 
**state** | **String** | The upload outcome: `created` (201), `duplicate` (200, an identical live asset) or `trashed` (200, an identical asset in the trash).  | 
**createdAt** | [**DateTime**](DateTime.md) | When the asset record was created (RFC 3339 UTC). | 
**takenAt** | [**DateTime**](DateTime.md) | Capture time (RFC 3339). Seeded from `deviceCreatedAt` (or the upload time when absent) and refined from EXIF by the metadata pipeline; responses render the best value the server has at answer time.  | 
**isTrashed** | **bool** | True when the asset sits in the trash; `state: trashed` responses carry it as true so clients can offer the restore or save-as-new decision.  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


