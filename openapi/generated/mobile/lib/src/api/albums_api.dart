//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

import 'dart:async';

import 'package:built_value/json_object.dart';
import 'package:built_value/serializer.dart';
import 'package:dio/dio.dart';

import 'package:lumemor_api/src/api_util.dart';
import 'package:lumemor_api/src/model/album_detail.dart';
import 'package:lumemor_api/src/model/album_wall_response.dart';
import 'package:lumemor_api/src/model/error.dart';

class AlbumsApi {

  final Dio _dio;

  final Serializers _serializers;

  const AlbumsApi(this._dio, this._serializers);

  /// One album&#39;s detail and one page of its items.
  /// The detail view of one album the caller can see: the caller owns it, or it is shared to the caller through an &#x60;album_shares&#x60; row. &#x60;callerRole&#x60; says which -- &#x60;owner&#x60; for the owner&#39;s write affordances, &#x60;viewer&#x60; for the read-only share.  Sync albums are not album resources: the id of a sync album -- the caller&#39;s own or anyone else&#39;s -- answers the same 404 as an unknown id. The sync album is the timeline (&#x60;GET /assets/timeline&#x60;); its stream surfaces there, never through this face.  404 is single-shape by design: an unknown id, another account&#39;s unshared album, and a sync album all answer the same &#x60;ALBUM_NOT_FOUND&#x60; byte for byte -- the anti-enumeration discipline of &#x60;ASSET_NOT_FOUND&#x60; applied to the album face. An album whose share to the caller was revoked before the request simply sits in that same invisible region.  &#x60;itemCount&#x60; counts the album&#39;s live members; &#x60;items&#x60; is one keyset page of them, ordered &#x60;takenAt&#x60; descending with the asset id as the tiebreak -- the timeline&#39;s own ordering and cursor pair, so the album grid pages with the timeline&#39;s walking machinery. Every item carries the membership&#39;s &#x60;addedAt&#x60; and the minimal asset context the grid renders; &#x60;thumbnail&#x60; is the asset thumbnail endpoint&#39;s &#x60;tile&#x60;-size URL. Live members only: trashed, hidden and Live Photo component rows never appear (A5 -- a hidden asset is invisible to the whole family).  &#x60;coverAssetId&#x60; is the raw pinned column: the single asset the owner chose, or null when unset -- the wall derives its collage fallback from the members, the detail carries the pin itself.  &#x60;limit&#x60; bounds the page size: absent means the default of 60; a value below 1 or over 200 answers 400 &#x60;VALIDATION_FAILED&#x60;. Pages continue through the opaque &#x60;cursor&#x60;, the timeline&#39;s exact keyset semantics: &#x60;nextCursor&#x60; is present exactly when another page follows, and the last page carries none. 
  ///
  /// Parameters:
  /// * [albumId] - Identifier of the album.
  /// * [cursor] - The `nextCursor` of the previous page; absent on the first page. An unparseable value answers 400 `VALIDATION_FAILED` (cursors are opaque strings produced by this endpoint, never assembled by clients). 
  /// * [limit] - Page size: the number of album items one response carries at most. Defaults to 60; values below 1 or over 200 answer 400. 
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [AlbumDetail] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<AlbumDetail>> getAlbum({ 
    required String albumId,
    String? cursor,
    int? limit = 60,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/albums/{albumId}'.replaceAll('{' r'albumId' '}', encodeQueryParameter(_serializers, albumId, const FullType(String)).toString());
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

    AlbumDetail? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(AlbumDetail),
      ) as AlbumDetail;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<AlbumDetail>(
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

  /// The caller&#39;s album wall.
  /// The album wall of the calling user: every album that deserves a wall card, in two groups concatenated into one array -- the web client renders its sections off the &#x60;callerRole&#x60; discriminator each item carries.  The own group comes first: the albums the caller owns of type &#x60;virtual&#x60; or &#x60;directory&#x60;, the most recently updated first (the album id descending as the tiebreak). The shared group follows: every album shared to the caller through an &#x60;album_shares&#x60; row, the caller&#39;s most recently granted share first (the album id descending as the tiebreak). Shared items carry &#x60;owner&#x60; -- the honest attribution of another account&#39;s album: the owner&#39;s id and email.  Sync albums NEVER appear on the wall, in either group. The sync album is the account&#39;s core backup stream and lives as the timeline (&#x60;GET /assets/timeline&#x60;), not as an album resource; a sync album another account shared is likewise no wall card. The caller&#39;s own sync album is invisible here and answers 404 through &#x60;GET /albums/{albumId}&#x60;.  &#x60;coverAssetIds&#x60; is the wall card&#39;s collage source: the pinned &#x60;coverAssetId&#x60; as a single-element list when the owner set one, otherwise the album&#39;s most recent live member assets -- up to four, &#x60;takenAt&#x60; descending with the asset id as the tiebreak, fewer when the album holds fewer members, empty for an album with none. &#x60;itemCount&#x60; counts live members under the same predicate: trashed, hidden and Live Photo component rows never count (A5 -- a hidden asset is invisible to the whole family).  &#x60;isShared&#x60; reports whether the album carries at least one &#x60;album_shares&#x60; row -- the owner&#39;s \&quot;someone else sees this\&quot; badge. It is true by construction on every shared-group item (the caller&#39;s share row is the reason the item exists). 
  ///
  /// Parameters:
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [AlbumWallResponse] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<AlbumWallResponse>> listAlbums({ 
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/albums';
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

    AlbumWallResponse? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(AlbumWallResponse),
      ) as AlbumWallResponse;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<AlbumWallResponse>(
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
