# lumemor_api.model.AlbumOwnerRef

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**id** | **String** | Identifier of the owning account. | 
**email** | **String** | The email address of the owning account: the fallback the client renders when `name` is empty, and the disambiguator between two accounts that share a name.  | 
**name** | **String** | The owning account's display name, exactly as stored. The empty string when the account set none -- the client then renders `email` instead. Never treated as a unique key: two accounts may share a name, and the email is what tells them apart.  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


