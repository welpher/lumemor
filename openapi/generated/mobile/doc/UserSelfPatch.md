# lumemor_api.model.UserSelfPatch

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**name** | **String** | The new display name; 1 to 100 runes after trimming (an empty name is refused -- the profile always carries one).  | [optional] 
**email** | **String** | The new login address, parsed and lower-cased like every creation path; an address another account already owns answers 409 USER_EMAIL_TAKEN.  | [optional] 
**theme** | [**UserTheme**](UserTheme.md) |  | [optional] 
**locale** | [**UserLocale**](UserLocale.md) |  | [optional] 
**downloadFormat** | [**DownloadFormat**](DownloadFormat.md) |  | [optional] 
**downloadIncludeVideos** | **bool** | The download packaging's video switch. | [optional] 
**downloadIncludeSidecars** | **bool** | The download packaging's sidecar switch. | [optional] 
**downloadIncludeEmbeddedVideos** | **bool** | The download packaging's embedded-motion switch (default false: the single-file motion photo's original already carries its motion).  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


