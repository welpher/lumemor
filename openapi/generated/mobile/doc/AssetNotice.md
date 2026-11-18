# lumemor_api.model.AssetNotice

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**code** | **String** | What happened.  `LIVE_PHOTO_MOTION_ABSORBED`: the Live Photo's motion bytes already existed as a live standalone video of this owner; that row became the pair's motion (`assetId` names it) and no longer lists as its own asset. Carried on a 201 `created` answer.  `LIVE_PHOTO_MOTION_ALREADY_STORED`: the uploaded bytes already play as a Live Photo's motion, so the returned row is that hidden component (`assetId` names it) and does not list on its own. Carried on a 200 `duplicate` answer: the content genuinely exists, so the state stays `duplicate` -- there is nothing for the user to fix.  | 
**assetId** | **String** | The asset the notice is about: the video row that plays as a Live Photo's motion.  | 
**name** | **String** | The affected asset's original filename, so the message can name what moved.  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


