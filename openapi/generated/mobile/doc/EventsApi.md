# lumemor_api.api.EventsApi

## Load the API package
```dart
import 'package:lumemor_api/api.dart';
```

All URIs are relative to */api/v1*

Method | HTTP request | Description
------------- | ------------- | -------------
[**getEvents**](EventsApi.md#getevents) | **GET** /events | Stream instance activity over server-sent events.


# **getEvents**
> String getEvents(lastEventID)

Stream instance activity over server-sent events.

The activity stream of the instance, authenticated like the REST API: the web session cookie and the mobile bearer token both open the stream.  Every event is one `text/event-stream` block: `event:` carries the event name, `id:` a monotonically increasing number, and `data:` the JSON payload -- the resource id plus a summary of the changed fields. Event names follow `<resource>.<past-tense verb>`:  - `asset.uploaded`, `asset.ready`, `asset.updated`,   `asset.trashed`, `asset.restored`, `asset.deleted` - `album.created`, `album.updated`, `album.deleted`,   `album.items-changed`, `album.shared`, `album.unshared` - `auth.login_failed` - `face.clustered` - `job.started`, `job.progress`, `job.finished` - `pairing.token_created` - `person.labeled`, `person.unlabeled`, `person.hidden`,   `person.merged`, `person.face_moved` - `session.started`, `session.revoked` - `server.settings-changed`  The `person.*` events report the people surface's label, hidden, merge and face-move writes. Every payload carries the acting user's `userId` beside the cluster (or face) id, because label and hidden state are private per user: a subscriber filters by the ids it renders, never by another account's change. `person.merged` carries the dissolved source cluster ids comma-joined as `mergedClusterIds`, and `person.face_moved` carries the face's new `clusterId`.  The server keeps the last five minutes of events in a ring buffer. Browser EventSource clients reconnect automatically and resend `Last-Event-ID`; the server replays everything newer than that id, and events that fell out of the ring are skipped. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getEventsApi();
final String lastEventID = lastEventID_example; // String | The id of the last received event, resent automatically by browser EventSource implementations on reconnect. When present, the server replays the events newer than this id from its five-minute ring buffer; events that fell out of the ring are skipped. 

try {
    final response = api.getEvents(lastEventID);
    print(response);
} on DioException catch (e) {
    print('Exception when calling EventsApi->getEvents: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **lastEventID** | **String**| The id of the last received event, resent automatically by browser EventSource implementations on reconnect. When present, the server replays the events newer than this id from its five-minute ring buffer; events that fell out of the ring are skipped.  | [optional] 

### Return type

**String**

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: text/event-stream, application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

