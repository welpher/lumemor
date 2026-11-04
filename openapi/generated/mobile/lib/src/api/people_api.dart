//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

import 'dart:async';

import 'package:built_value/json_object.dart';
import 'package:built_value/serializer.dart';
import 'package:dio/dio.dart';

import 'package:built_collection/built_collection.dart';
import 'package:lumemor_api/src/api_util.dart';
import 'package:lumemor_api/src/model/cluster_detail.dart';
import 'package:lumemor_api/src/model/error.dart';
import 'package:lumemor_api/src/model/person_summary.dart';
import 'package:lumemor_api/src/model/unnamed_clusters_response.dart';

class PeopleApi {

  final Dio _dio;

  final Serializers _serializers;

  const PeopleApi(this._dio, this._serializers);

  /// One face cluster&#39;s detail as the caller sees it.
  /// The detail view of one face cluster: the caller&#39;s own label state and the cluster&#39;s faces on assets visible to the caller, strongest score first with the face id as the tiebreak.  &#x60;label&#x60; is null when the caller has not labeled the cluster (the unnamed state). It is the CALLING user&#39;s label or no label at all: another account&#39;s label never shows here (B4 -- clusters are global, labels are private per user).  404 has a double meaning by design: the cluster id is unknown, or the cluster exists but none of its faces sits on an asset visible to the caller (before library sharing: the cluster&#39;s assets all belong to other accounts). Both answer the same &#x60;PERSON_NOT_FOUND&#x60;, byte for byte, so cluster ids cannot be probed for existence -- the same anti-enumeration shape as &#x60;ASSET_NOT_FOUND&#x60;.  Every &#x60;thumbnail&#x60; points at the face crop endpoint &#x60;GET /api/v1/faces/{faceId}/thumbnail&#x60;, which serves the face&#39;s bounding-box crop of the asset&#39;s view-size derivative. 
  ///
  /// Parameters:
  /// * [clusterId] - Identifier of the face cluster.
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [ClusterDetail] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<ClusterDetail>> getPersonCluster({ 
    required String clusterId,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/people/{clusterId}'.replaceAll('{' r'clusterId' '}', encodeQueryParameter(_serializers, clusterId, const FullType(String)).toString());
    final _options = Options(
      method: r'GET',
      headers: <String, dynamic>{
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

    ClusterDetail? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(ClusterDetail),
      ) as ClusterDetail;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<ClusterDetail>(
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

  /// The caller&#39;s named people.
  /// The named region of the people surface: every face cluster the caller sees that carries one of the caller&#39;s own labels. A cluster is listed here when both hold:  - it contains at least one face on an asset visible to the   caller (before library sharing exists: an asset the caller   owns that is live -- not trashed, not hidden, no Live Photo   component row), and - the caller has labeled it with a &#x60;user_face_labels&#x60; row of   their own.  The B4 semantics of the people domain: face clusters are GLOBAL structures of the instance (the same person&#39;s faces land in one cluster whatever account uploaded the photos), while names and the hidden switch are PRIVATE per user -- two accounts label the same cluster independently and neither sees the other&#39;s label. &#x60;faceCount&#x60; is honest per caller: it counts only the cluster&#39;s faces on caller-visible assets, so the same cluster can show different counts to different accounts.  Hidden people STAY in this list carrying &#x60;isHidden&#x60; true: the switch is a badge for the web UI, not a filter -- the caller labeled the person and keeps seeing them, flagged.  The list is sorted by name, case-insensitively, with the cluster id as the tiebreak. 
  ///
  /// Parameters:
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [BuiltList<PersonSummary>] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<BuiltList<PersonSummary>>> listPeople({ 
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/people';
    final _options = Options(
      method: r'GET',
      headers: <String, dynamic>{
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

    BuiltList<PersonSummary>? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(BuiltList, [FullType(PersonSummary)]),
      ) as BuiltList<PersonSummary>;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<BuiltList<PersonSummary>>(
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

  /// The caller&#39;s unnamed face clusters, one keyset page at a time.
  /// The unnamed region of the people surface: every face cluster that contains at least one face on an asset visible to the caller (before library sharing exists: an asset the caller owns that is live -- not trashed, not hidden, no Live Photo component row) and carries NO &#x60;user_face_labels&#x60; row of the caller. The moment the caller labels a cluster it moves to &#x60;GET /people&#x60; and out of this list; another account&#39;s label never moves it for this caller (B4: clusters are global, labels are private per user).  Pages are keyset-paginated with an opaque cursor over the ordering &#x60;faceCount&#x60; descending, &#x60;clusterId&#x60; descending as the tiebreak. The continuation key is the (faceCount, clusterId) pair of a page&#39;s last row -- the same row-value predicate pattern the timeline endpoint applies to its (takenAt, id) pair, which keeps pages stable while clusters gain or lose faces. The response carries &#x60;nextCursor&#x60; exactly when another page exists; the first page omits the &#x60;cursor&#x60; parameter.  &#x60;limit&#x60; bounds the page size: absent means the default of 60; a value below 1 or over 200 answers 400 &#x60;VALIDATION_FAILED&#x60;. 
  ///
  /// Parameters:
  /// * [cursor] - The `nextCursor` of the previous page; absent on the first page. An unparseable value answers 400 `VALIDATION_FAILED` (cursors are opaque strings produced by this endpoint, never assembled by clients). 
  /// * [limit] - Page size: the number of cluster summaries one response carries at most. Defaults to 60; values below 1 or over 200 answer 400. 
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [UnnamedClustersResponse] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<UnnamedClustersResponse>> listUnnamedPeople({ 
    String? cursor,
    int? limit = 60,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/people/unnamed';
    final _options = Options(
      method: r'GET',
      headers: <String, dynamic>{
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

    final _queryParameters = <String, dynamic>{
      if (cursor != null) r'cursor': encodeQueryParameter(_serializers, cursor, const FullType(String)),
      if (limit != null) r'limit': encodeQueryParameter(_serializers, limit, const FullType(int)),
    };

    final _response = await _dio.request<Object>(
      _path,
      options: _options,
      queryParameters: _queryParameters,
      cancelToken: cancelToken,
      onSendProgress: onSendProgress,
      onReceiveProgress: onReceiveProgress,
    );

    UnnamedClustersResponse? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(UnnamedClustersResponse),
      ) as UnnamedClustersResponse;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<UnnamedClustersResponse>(
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
