# lumemor_api.model.UserSelfResponse

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**id** | **String** | UUIDv7 identifier of the account. | 
**name** | **String** | Display name. | 
**email** | **String** | Login address; unique instance-wide. | 
**role** | [**UserRole**](UserRole.md) |  | 
**theme** | [**UserTheme**](UserTheme.md) |  | 
**locale** | [**UserLocale**](UserLocale.md) |  | 
**downloadFormat** | [**DownloadFormat**](DownloadFormat.md) |  | 
**downloadIncludeVideos** | **bool** | Whether the account's download packaging includes the original video files. Photos are the download's content and always ride along.  | 
**downloadIncludeSidecars** | **bool** | Whether the account's download packaging includes EXIF sidecar files (`.xmp`) beside the media.  | 
**downloadIncludeEmbeddedVideos** | **bool** | Whether a download adds the motion a single-file \"motion photo\" embeds as a separate video file. Default `false`: the still's original already carries that motion, so the extra copy is a duplicate unless the account asks for it. A paired Live Photo's own motion file always rides along and this switch never removes it.  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


