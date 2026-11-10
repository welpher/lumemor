//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

import 'dart:async';

import 'package:built_value/json_object.dart';
import 'package:built_value/serializer.dart';
import 'package:dio/dio.dart';

import 'package:built_collection/built_collection.dart';
import 'package:lumemor_api/src/api_util.dart';
import 'package:lumemor_api/src/model/error.dart';
import 'package:lumemor_api/src/model/media_kind.dart';
import 'package:lumemor_api/src/model/search_response.dart';

class SearchApi {

  final Dio _dio;

  final Serializers _serializers;

  const SearchApi(this._dio, this._serializers);

  /// The unified search, one text box, two merged paths, filter facets.
  /// The unified search over the caller-visible asset domain (the album union of the caller&#39;s own sync and directory albums plus every album shared to them, and the timeline&#39;s live-row filters: not trashed, not hidden, no Live Photo component row). One optional text box drives TWO backend paths that merge deterministically; the optional filter parameters narrow both paths and the pure-filter mode (no text) is the filter browse the search page opens with.  The TEXT path is unaccented case-insensitive SUBSTRING matching over the asset&#39;s own text sources -- the original filename, the description, and the extracted OCR text -- plus the caller&#39;s relational vocabularies joined at query time: the caller&#39;s own tags, the titles of albums the caller can see (own albums and albums shared to them; the sync album is the timeline&#39;s face, not a filter), and the caller&#39;s own non-hidden person labels. The query is a LITERAL substring, never a pattern: the engine escapes its own wildcard characters, so &#x60;50%&#x60; matches the literal text &#x60;50%&#x60; and never globs. Word semantics, stemming and segmentation are out of scope by design (the v1 boundary: the match is containment, which is language-agnostic by construction; a tokenizing engine is the v1.x upgrade path). The path&#39;s weak rank signal is the trigram similarity of the best matched source.  The SEMANTIC path embeds the query through the active CLIP text tower (the model registry&#39;s clip-text slot) and ranks the caller-visible assets by cosine similarity over the image embeddings the active clip-embed model produced -- the same joint space both towers share. The semantic path needs the model machinery: when no text tower is active (or the tower&#39;s dimension disagrees with the embeddings column) the semantic path honestly answers EMPTY and the text path still serves -- the degradation the model admin page resolves by downloading and activating the text tower.  The MERGE: both paths answer at most their top 500 ranked hits; an asset&#39;s fused score is the sum over the paths it appears in of (500 minus its zero-based position in that path), so an asset hit by both paths outranks the single-path hits around it; the page orders by the fused score descending with the asset id ascending as the total-order tiebreak, and one asset hit through many sources or both paths appears exactly once. The rule is deterministic: the same query over unchanged data answers the same page, and cursors page through that stable order.  A query of fewer than two characters (after trimming), or only whitespace, takes NO text path and NO semantic path: the request is the pure filter browse. An empty &#x60;from&#x60;/&#x60;to&#x60; pair with no other filter is the whole visible timeline.  The FACETS count the current filtered set minus their own dimension: &#x60;people&#x60; lists the caller&#39;s named, non-hidden people with the number of assets of the current set (query and every filter applied EXCEPT personIds) that carry a face of that cluster; &#x60;places&#x60;, &#x60;years&#x60;, &#x60;cameras&#x60;, &#x60;kinds&#x60;, &#x60;albums&#x60; and &#x60;tags&#x60; follow the same rule for their own dimension. Facet ids the caller cannot resolve (an album they cannot see, another account&#39;s tag) DROP OUT of the filter parameters rather than emptying the result -- filters narrow, they never probe.  &#x60;personIds&#x60;, &#x60;albumIds&#x60; and &#x60;tagIds&#x60; are comma-separated id lists; &#x60;placeIds&#x60; is a comma-separated integer list; &#x60;cameras&#x60; is a comma-separated list of the camera facet ids (&#x60;make|model&#x60;). Every id list is capped at 200 entries; a longer list answers 400 &#x60;VALIDATION_FAILED&#x60;. &#x60;from&#x60; is inclusive, &#x60;to&#x60; is EXCLUSIVE; an inverted range answers 400. &#x60;locale&#x60; picks the places facet&#39;s display names exactly the way &#x60;GET /places&#x60; does. 
  ///
  /// Parameters:
  /// * [query] - The free-text query: a literal substring matched (unaccented, case-insensitively) against the filename, the description, the OCR text and the caller's tag, album-title and person-name vocabularies, and embedded through the CLIP text tower for the semantic path. A value of fewer than two characters (after trimming) takes no text path: the request is the pure filter browse. The length is capped at 256 characters. 
  /// * [personIds] - Restrict the results to assets carrying a face of one of these face clusters (the caller's people surface's cluster ids). 
  /// * [placeIds] - Restrict the results to assets resolved to one of these places (the places facet's GeoNames ids). 
  /// * [from] - The inclusive lower bound of the taken-at range. 
  /// * [to] - The EXCLUSIVE upper bound of the taken-at range. A `to` at or before `from` answers 400 `VALIDATION_FAILED`. 
  /// * [cameras] - Restrict the results to assets whose EXIF camera matches one of these camera facet ids (`make|model`). 
  /// * [kinds] - Restrict the results to these media families. A value outside `image`/`video` answers 400. 
  /// * [albumIds] - Restrict the results to members of these albums. Albums the caller cannot see drop out of the filter: they contribute nothing, they never empty the result. 
  /// * [tagIds] - Restrict the results to assets carrying one of the CALLER's tags. Another account's tag ids drop out of the filter (tags are per-user state). 
  /// * [locale] - The display locale of the places facet's names, exactly the `GET /places` locale join. Absent means `en`. A value outside the set answers 400 `VALIDATION_FAILED`. 
  /// * [cursor] - The `nextCursor` of the previous page; absent on the first page. The cursor carries the page's ordering mode (the fused rank of a text query, or the timeline order of the filter browse): a cursor of the other mode answers 400 `VALIDATION_FAILED`, and so does any value this endpoint did not produce. 
  /// * [limit] - Page size: the number of items one response carries at most. Defaults to 100; values over 200 are answered with 200, values below 1 answer 400. 
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [SearchResponse] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<SearchResponse>> searchAssets({ 
    String? query,
    BuiltList<String>? personIds,
    BuiltList<int>? placeIds,
    DateTime? from,
    DateTime? to,
    BuiltList<String>? cameras,
    BuiltList<MediaKind>? kinds,
    BuiltList<String>? albumIds,
    BuiltList<String>? tagIds,
    String? locale = 'en',
    String? cursor,
    int? limit = 100,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/search';
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
      if (query != null) r'query': encodeQueryParameter(_serializers, query, const FullType(String)),
      if (personIds != null) r'personIds': encodeCollectionQueryParameter<String>(_serializers, personIds, const FullType(BuiltList, [FullType(String)]), format: ListFormat.csv,),
      if (placeIds != null) r'placeIds': encodeCollectionQueryParameter<int>(_serializers, placeIds, const FullType(BuiltList, [FullType(int)]), format: ListFormat.csv,),
      if (from != null) r'from': encodeQueryParameter(_serializers, from, const FullType(DateTime)),
      if (to != null) r'to': encodeQueryParameter(_serializers, to, const FullType(DateTime)),
      if (cameras != null) r'cameras': encodeCollectionQueryParameter<String>(_serializers, cameras, const FullType(BuiltList, [FullType(String)]), format: ListFormat.csv,),
      if (kinds != null) r'kinds': encodeCollectionQueryParameter<MediaKind>(_serializers, kinds, const FullType(BuiltList, [FullType(MediaKind)]), format: ListFormat.csv,),
      if (albumIds != null) r'albumIds': encodeCollectionQueryParameter<String>(_serializers, albumIds, const FullType(BuiltList, [FullType(String)]), format: ListFormat.csv,),
      if (tagIds != null) r'tagIds': encodeCollectionQueryParameter<String>(_serializers, tagIds, const FullType(BuiltList, [FullType(String)]), format: ListFormat.csv,),
      if (locale != null) r'locale': encodeQueryParameter(_serializers, locale, const FullType(String)),
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

    SearchResponse? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(SearchResponse),
      ) as SearchResponse;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<SearchResponse>(
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
