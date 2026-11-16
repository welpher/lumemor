//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

import 'dart:async';

import 'package:built_value/json_object.dart';
import 'package:built_value/serializer.dart';
import 'package:dio/dio.dart';

import 'package:built_collection/built_collection.dart';
import 'package:lumemor_api/src/api_util.dart';
import 'package:lumemor_api/src/model/add_tag_assets_response.dart';
import 'package:lumemor_api/src/model/error.dart';
import 'package:lumemor_api/src/model/remove_tag_assets_response.dart';
import 'package:lumemor_api/src/model/tag_assets_request.dart';
import 'package:lumemor_api/src/model/tag_name_request.dart';
import 'package:lumemor_api/src/model/tag_summary.dart';

class TagsApi {

  final Dio _dio;

  final Serializers _serializers;

  const TagsApi(this._dio, this._serializers);

  /// Tag assets in one batch.
  /// Adds the caller&#39;s tag to a list of at most 200 asset ids with the batch surface&#39;s set semantics: one request, one honest report &#x60;{added, skipped}&#x60;.  Skip semantics (the missing-ids discipline, the album-items precedent widened to the union): every id must address an asset the CALLER CAN SEE -- the album union (the caller&#39;s own sync and directory albums plus every album shared to them), the same resolution the favorites arms of the batch endpoint apply. The union deliberately carries tags: a sharee tags what they see with THEIR OWN tag, and the owner&#39;s tag on the same asset stays untouched (the assignment rows are per-user). An unknown id, another account&#39;s unshared asset, and any id the caller cannot see are silently skipped, never errors. An id that already carries this tag is skipped too: assignment is a set, the add is idempotent. NO live-state filter applies to the resolution: the assignment row is per-user state written whatever the asset&#39;s live state (a trashed or archived asset carries the tag underneath the view&#39;s filters and the count returns with the restore or unhide, exactly like a favorite row). An all-skipped request still answers 200 with &#x60;added&#x60; 0. Repeated ids collapse to one application; the counts cover the distinct ids.  The tag must be the caller&#39;s own: an unknown id and another account&#39;s tag answer the same 404 &#x60;TAG_NOT_FOUND&#x60;. The write publishes nothing: the only view that changes is the caller&#39;s own tags, which the response itself reports. 
  ///
  /// Parameters:
  /// * [tagId] - Identifier of the tag.
  /// * [tagAssetsRequest] 
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [AddTagAssetsResponse] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<AddTagAssetsResponse>> addTagAssets({ 
    required String tagId,
    required TagAssetsRequest tagAssetsRequest,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/tags/{tagId}/assets'.replaceAll('{' r'tagId' '}', encodeQueryParameter(_serializers, tagId, const FullType(String)).toString());
    final _options = Options(
      method: r'POST',
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
      contentType: 'application/json',
      validateStatus: validateStatus,
    );

    dynamic _bodyData;

    try {
      const _type = FullType(TagAssetsRequest);
      _bodyData = _serializers.serialize(tagAssetsRequest, specifiedType: _type);

    } catch(error, stackTrace) {
      throw DioException(
         requestOptions: _options.compose(
          _dio.options,
          _path,
        ),
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    final _response = await _dio.request<Object>(
      _path,
      data: _bodyData,
      options: _options,
      cancelToken: cancelToken,
      onSendProgress: onSendProgress,
      onReceiveProgress: onReceiveProgress,
    );

    AddTagAssetsResponse? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(AddTagAssetsResponse),
      ) as AddTagAssetsResponse;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<AddTagAssetsResponse>(
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

  /// Create one tag, idempotently.
  /// Creates one tag owned by the caller. The name is trimmed server-side and must span 1-64 Unicode characters after the trim (the count is over runes, so multi-byte names stay within the bound at the same length); anything else answers 400 &#x60;VALIDATION_FAILED&#x60;.  Creation is IDEMPOTENT, the assignment face&#39;s set semantics applied to the vocabulary: a name the caller already owns converges on the tag that already exists. A fresh name answers 201 with the new row; a name that already exists answers 200 with the EXISTING tag -- the same id, nothing written, &#x60;updatedAt&#x60; untouched -- never a 409, because the vocabulary is a set and the duplicate ask is the same ask. Two concurrent creators converge on one row the same way (the unique constraint is the race guard). Names are distinct when their characters differ; the duplicate check is the constraint&#39;s own equality.  The response is the tag&#39;s summary exactly as &#x60;GET /tags&#x60; lists it. The write publishes nothing: tags are per-user state, the only view that changes is the caller&#39;s own, and the response itself carries it. 
  ///
  /// Parameters:
  /// * [tagNameRequest] 
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [TagSummary] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<TagSummary>> createTag({ 
    required TagNameRequest tagNameRequest,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/tags';
    final _options = Options(
      method: r'POST',
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
      contentType: 'application/json',
      validateStatus: validateStatus,
    );

    dynamic _bodyData;

    try {
      const _type = FullType(TagNameRequest);
      _bodyData = _serializers.serialize(tagNameRequest, specifiedType: _type);

    } catch(error, stackTrace) {
      throw DioException(
         requestOptions: _options.compose(
          _dio.options,
          _path,
        ),
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    final _response = await _dio.request<Object>(
      _path,
      data: _bodyData,
      options: _options,
      cancelToken: cancelToken,
      onSendProgress: onSendProgress,
      onReceiveProgress: onReceiveProgress,
    );

    TagSummary? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(TagSummary),
      ) as TagSummary;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<TagSummary>(
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

  /// Delete one tag.
  /// Deletes the caller&#39;s tag and every assignment row with it: the schema&#39;s &#x60;ON DELETE CASCADE&#x60; removes the asset_tags rows in the same statement, so a deleted tag leaves no orphans behind -- on any account&#39;s view, because the rows were the caller&#39;s own to begin with. The answer is 204 with no body.  Repeating the request answers 404 &#x60;TAG_NOT_FOUND&#x60;: the tag is gone exactly like an unknown id (the album delete&#39;s discipline -- a resource delete, not a set removal). The tag must be the caller&#39;s own; another account&#39;s tag folds into the same single-shape 404. The write publishes nothing (per-user state, the create&#39;s reasoning). 
  ///
  /// Parameters:
  /// * [tagId] - Identifier of the tag.
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future]
  /// Throws [DioException] if API call or serialization fails
  Future<Response<void>> deleteTag({ 
    required String tagId,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/tags/{tagId}'.replaceAll('{' r'tagId' '}', encodeQueryParameter(_serializers, tagId, const FullType(String)).toString());
    final _options = Options(
      method: r'DELETE',
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

    return _response;
  }

  /// The caller&#39;s tags, the search vocabulary.
  /// Every tag the calling user owns, one array sorted by name case-insensitively with the tag id as the deterministic tiebreak (the named-people listing&#39;s own ordering discipline): the vocabulary the search&#39;s tag facet, the batch bar&#39;s \&quot;add tag\&quot; and the info panel&#39;s tag section consume. Unpaginated like the places listing -- a family instance&#39;s vocabulary is a bounded set, one document.  Tags are per-user state (the favorites&#39; privacy shape): the tags&#39; owner is always the caller, and no tag, name or count of another account ever surfaces here. The listing carries the vocabulary in full -- a tag whose every member is trashed or archived still lists, with its honest count of zero (the assignment rows survive, exactly like a favorite row, and the count returns with the restore or unhide).  &#x60;assetCount&#x60; counts the tag&#39;s assignment rows joined to CALLER-VISIBLE LIVE assets: the album union (the caller&#39;s own sync and directory albums plus every album shared to them) and the timeline&#39;s live filters -- not trashed, not archived, no Live Photo component -- the honest-counts discipline of the people and places surfaces. A sharee&#39;s count covers the shared members they can see; a revoked share retracts them while the assignment rows survive underneath. 
  ///
  /// Parameters:
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [BuiltList<TagSummary>] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<BuiltList<TagSummary>>> listTags({ 
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/tags';
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

    BuiltList<TagSummary>? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(BuiltList, [FullType(TagSummary)]),
      ) as BuiltList<TagSummary>;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<BuiltList<TagSummary>>(
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

  /// Untag assets in one batch.
  /// Removes the caller&#39;s assignment rows of one tag: &#x60;{removed, skipped}&#x60;, the same batch semantics as the add. An id that carries no assignment of this tag -- never was, or already removed -- is skipped; an all-skipped request still answers 200 with &#x60;removed&#x60; 0. Repeated ids collapse; the counts cover the distinct ids.  The removal carries no visibility predicate: it tidies the caller&#39;s own set, so the rows leave whatever the assets&#39; live state (a trashed member&#39;s assignment row still leaves, the honest cleanup of the set itself). The tag must be the caller&#39;s own: an unknown id and another account&#39;s tag answer the same 404 &#x60;TAG_NOT_FOUND&#x60;. The write publishes nothing (per-user state, the add&#39;s reasoning). 
  ///
  /// Parameters:
  /// * [tagId] - Identifier of the tag.
  /// * [tagAssetsRequest] 
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [RemoveTagAssetsResponse] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<RemoveTagAssetsResponse>> removeTagAssets({ 
    required String tagId,
    required TagAssetsRequest tagAssetsRequest,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/tags/{tagId}/assets'.replaceAll('{' r'tagId' '}', encodeQueryParameter(_serializers, tagId, const FullType(String)).toString());
    final _options = Options(
      method: r'DELETE',
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
      contentType: 'application/json',
      validateStatus: validateStatus,
    );

    dynamic _bodyData;

    try {
      const _type = FullType(TagAssetsRequest);
      _bodyData = _serializers.serialize(tagAssetsRequest, specifiedType: _type);

    } catch(error, stackTrace) {
      throw DioException(
         requestOptions: _options.compose(
          _dio.options,
          _path,
        ),
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    final _response = await _dio.request<Object>(
      _path,
      data: _bodyData,
      options: _options,
      cancelToken: cancelToken,
      onSendProgress: onSendProgress,
      onReceiveProgress: onReceiveProgress,
    );

    RemoveTagAssetsResponse? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(RemoveTagAssetsResponse),
      ) as RemoveTagAssetsResponse;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<RemoveTagAssetsResponse>(
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

  /// Rename one tag.
  /// Renames the caller&#39;s tag. The name validation is the create&#39;s own: trimmed server-side, 1-64 Unicode characters after the trim, else 400 &#x60;VALIDATION_FAILED&#x60;.  Renaming to the name the row already carries is the idempotent no-op: 200 with the row&#39;s current state, nothing written (not even &#x60;updatedAt&#x60;). Renaming to a name ANOTHER of the caller&#39;s tags carries answers 409 &#x60;TAG_NAME_TAKEN&#x60;: the vocabulary keeps one row per name, and the honest answer for a collision is the conflict, not a silent merge into the other tag. (The create&#39;s idempotence does not extend here: the URL already names one tag, and converging on the other row would substitute the resource instead of answering.)  The response is the tag&#39;s summary after the edit, exactly as &#x60;GET /tags&#x60; lists it. The write publishes nothing (per-user state, the create&#39;s reasoning). The tag must be the caller&#39;s own: an unknown id and another account&#39;s tag answer the same 404 &#x60;TAG_NOT_FOUND&#x60;, byte for byte, so tag ids cannot be probed for existence. 
  ///
  /// Parameters:
  /// * [tagId] - Identifier of the tag.
  /// * [tagNameRequest] 
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [TagSummary] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<TagSummary>> renameTag({ 
    required String tagId,
    required TagNameRequest tagNameRequest,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/tags/{tagId}'.replaceAll('{' r'tagId' '}', encodeQueryParameter(_serializers, tagId, const FullType(String)).toString());
    final _options = Options(
      method: r'PATCH',
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
      contentType: 'application/json',
      validateStatus: validateStatus,
    );

    dynamic _bodyData;

    try {
      const _type = FullType(TagNameRequest);
      _bodyData = _serializers.serialize(tagNameRequest, specifiedType: _type);

    } catch(error, stackTrace) {
      throw DioException(
         requestOptions: _options.compose(
          _dio.options,
          _path,
        ),
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    final _response = await _dio.request<Object>(
      _path,
      data: _bodyData,
      options: _options,
      cancelToken: cancelToken,
      onSendProgress: onSendProgress,
      onReceiveProgress: onReceiveProgress,
    );

    TagSummary? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(TagSummary),
      ) as TagSummary;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<TagSummary>(
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
