# lumemor_api.model.SessionInfo

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**id** | **String** | UUIDv7 identifier of the session. | 
**kind** | **String** | Where the session credential lives: `web` sessions ride the `lumemor_session` cookie and slide their expiry by 30 days on use; `mobile` sessions ride a bearer token and are revoked explicitly.  | 
**deviceName** | **String** | Human-readable device label of the session. For web sessions it is the browser user agent captured at login; mobile sessions carry the name supplied during pairing.  | [optional] 
**createdAt** | [**DateTime**](DateTime.md) | When the session was created (RFC 3339 UTC). | 
**lastSeenAt** | [**DateTime**](DateTime.md) | When the session was last used (RFC 3339 UTC). | 
**expiresAt** | [**DateTime**](DateTime.md) | When the session expires (RFC 3339 UTC). Null for bearer sessions, which are revoked explicitly instead of expiring.  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


