# lumemor_api.model.ModelCatalogEntry

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**id** | **String** | The manifest model's stable id. | 
**slot** | [**ModelSlot**](ModelSlot.md) |  | 
**name** | **String** | The display name the manifest's notice carries, derived from the model id. | 
**license** | **String** | The upstream license identifier, empty when undeclared. | 
**dim** | **int** | The embedding dimension; absent for models that do not embed. | [optional] 
**bytes** | **int** | The pinned byte count of the entry's files. | 
**downloaded** | **bool** | Whether the entry's files are present on disk (a presence probe, not a digest check). | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


