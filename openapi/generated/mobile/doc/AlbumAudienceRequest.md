# lumemor_api.model.AlbumAudienceRequest

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**audience** | **BuiltList&lt;String&gt;** | Every account that may view the album after this write, between 0 and 200 entries. Repeated ids collapse to one application. The owner's own id is skippable noise (the owner sees the album by construction), and ids that name no usable account are skipped with a reason -- never errors.  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


