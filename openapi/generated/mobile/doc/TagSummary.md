# lumemor_api.model.TagSummary

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**id** | **String** | UUIDv7 identifier of the tag (the caller's own row). | 
**name** | **String** | The tag's display name, unique within the caller's vocabulary. Trimmed on write; 1-64 Unicode characters (counted over runes).  | 
**assetCount** | **int** | The tag's assignment rows joined to caller-visible live assets: the album union plus the timeline's live filters (not trashed, not archived, no Live Photo component). A trashed or archived member stops counting until it returns; the assignment row survives underneath, exactly like a favorite row.  | 
**createdAt** | [**DateTime**](DateTime.md) | When the tag row was created (RFC 3339 UTC). | 
**updatedAt** | [**DateTime**](DateTime.md) | When the tag's name last truly changed (RFC 3339 UTC). The idempotent create and the no-op rename leave it untouched -- only a real rename moves it.  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


