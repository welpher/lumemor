# lumemor_api.model.AssetPatch

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**description** | **String** | The owner's free-text description of the asset, the text the detail panel renders. The value replaces the stored text wholesale (an empty string clears it). At most 10,000 Unicode characters -- the count is over runes, so a text of multi-byte characters stays within the bound at the same length.  | [optional] 
**isHidden** | **bool** | The owner-level visibility switch: `true` keeps the asset out of every live view (the timeline, the favorites) until it is unhidden; `false` returns it. The same switch the batch `hide` / `unhide` actions flip.  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


