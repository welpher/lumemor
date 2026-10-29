//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

import 'dart:async';

import 'package:built_value/json_object.dart';
import 'package:built_value/serializer.dart';
import 'package:dio/dio.dart';

import 'package:lumemor_api/src/model/error.dart';

class EventsApi {

  final Dio _dio;

  final Serializers _serializers;

  const EventsApi(this._dio, this._serializers);

  /// Stream instance activity over server-sent events.
  /// The activity stream of the instance, authenticated like the REST API: the web session cookie and the mobile bearer token both open the stream.  Every event is one &#x60;text/event-stream&#x60; block: &#x60;event:&#x60; carries the event name, &#x60;id:&#x60; a monotonically increasing number, and &#x60;data:&#x60; the JSON payload -- the resource id plus a summary of the changed fields. Event names follow &#x60;&lt;resource&gt;.&lt;past-tense verb&gt;&#x60;:  - &#x60;asset.uploaded&#x60;, &#x60;asset.ready&#x60;, &#x60;asset.updated&#x60;,   &#x60;asset.trashed&#x60;, &#x60;asset.restored&#x60;, &#x60;asset.deleted&#x60; - &#x60;album.created&#x60;, &#x60;album.updated&#x60;, &#x60;album.deleted&#x60;,   &#x60;album.items-changed&#x60;, &#x60;album.shared&#x60;, &#x60;album.unshared&#x60; - &#x60;auth.login_failed&#x60; - &#x60;face.cluster-updated&#x60; - &#x60;job.progress&#x60;, &#x60;job.finished&#x60; - &#x60;model.changed&#x60; - &#x60;pairing.token_created&#x60; - &#x60;session.started&#x60;, &#x60;session.revoked&#x60; - &#x60;server.settings-changed&#x60;  The server keeps the last five minutes of events in a ring buffer. Browser EventSource clients reconnect automatically and resend &#x60;Last-Event-ID&#x60;; the server replays everything newer than that id, and events that fell out of the ring are skipped. 
  ///
  /// Parameters:
  /// * [lastEventID] - The id of the last received event, resent automatically by browser EventSource implementations on reconnect. When present, the server replays the events newer than this id from its five-minute ring buffer; events that fell out of the ring are skipped. 
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [String] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<String>> getEvents({ 
    String? lastEventID,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/events';
    final _options = Options(
      method: r'GET',
      headers: <String, dynamic>{
        if (lastEventID != null) r'Last-Event-ID': lastEventID,
        ...?headers,
      },
      extra: <String, dynamic>{
        'secure': <Map<String, String>>[
          {
            'type': 'apiKey',
            'name': 'cookieAuth',
            'keyName': 'lumemor_session',
            'where': '',
          },{
            'type': 'http',
            'scheme': 'bearer',
            'name': 'bearerAuth',
          },
        ],
        ...?extra,
      },
      validateStatus: validateStatus,
    );

    final _response = await _dio.request<Object>(
      _path,
      options: _options,
      cancelToken: cancelToken,
      onSendProgress: onSendProgress,
      onReceiveProgress: onReceiveProgress,
    );

    String? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : rawResponse as String;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<String>(
      data: _responseData,
      headers: _response.headers,
      isRedirect: _response.isRedirect,
      requestOptions: _response.requestOptions,
      redirects: _response.redirects,
      statusCode: _response.statusCode,
      statusMessage: _response.statusMessage,
      extra: _response.extra,
    );
  }

}
