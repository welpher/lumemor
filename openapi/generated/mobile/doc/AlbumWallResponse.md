# lumemor_api.model.AlbumWallResponse

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**albums** | [**BuiltList&lt;AlbumSummary&gt;**](AlbumSummary.md) | Every wall album: the own group first (the caller's virtual and directory albums, `updatedAt` descending), then the shared group (the albums shared to the caller, the caller's grant time descending). Each group breaks ties on the album id, descending. Sync albums never appear in either group.  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


