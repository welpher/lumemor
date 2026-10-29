# lumemor_api.model.PairingTokenResponse

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**token** | **String** | One-time pairing secret, 128 random bits as a URL-safe string. Render it as a QR code and never store or log it; the server keeps only its hash.  | 
**expiresAt** | [**DateTime**](DateTime.md) | When the token expires (RFC 3339 UTC); five minutes after issuance.  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


