# lumemor_api.model.AssetNotice

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**code** | **String** | What happened. `LIVE_PHOTO_MOTION_ABSORBED`: the Live Photo's motion bytes already existed as a live standalone video of this owner; that row became the pair's motion (`assetId` names it) and no longer lists as its own asset.  | 
**assetId** | **String** | The asset the notice is about: the reused row that became the Live Photo's motion.  | 
**name** | **String** | The affected asset's original filename, so the message can name what moved.  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


