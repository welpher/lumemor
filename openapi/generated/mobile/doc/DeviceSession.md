# lumemor_api.model.DeviceSession

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**id** | **String** | UUIDv7 identifier of the session. | 
**kind** | **String** | Where the session credential lives: `web` sessions ride the `lumemor_session` cookie, `mobile` sessions ride a bearer token.  | 
**deviceName** | **String** | Human-readable device label: the browser's user agent captured at login for web sessions, the name supplied during pairing for mobile ones.  | [optional] 
**lastSeenAt** | [**DateTime**](DateTime.md) | When the session was last used (RFC 3339 UTC). | 
**ipAddress** | **String** | The address the session was created from (the login's or pairing exchange's socket peer), captured at creation time. Omitted when the row predates the column or no address was captured; an audit hint, not a geolocation claim.  | [optional] 
**isCurrent** | **bool** | Whether this row is the session the request itself authenticates with -- the device the caller is managing from. Exactly one entry of a list carries `true`, and that entry cannot be revoked through `DELETE /users/me/sessions/{id}` (the sign-out flow is its exit).  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


