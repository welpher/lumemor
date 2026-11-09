# lumemor_api.model.AlbumAudienceSkip

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**userId** | **String** | The requested id that stayed ungranted. | 
**reason** | **String** | Why the id was skipped: `owner` -- the album's own owner, whose visibility is the ownership row, never a share row; `invalid_account` -- an unknown or disabled account (the same reason for both, so user ids cannot be probed).  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


