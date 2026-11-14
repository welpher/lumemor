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
import 'package:lumemor_api/src/model/map_config_response.dart';
import 'package:lumemor_api/src/model/place_summary.dart';

class PlacesApi {

  final Dio _dio;

  final Serializers _serializers;

  const PlacesApi(this._dio, this._serializers);

  /// The map&#39;s client-side tile configuration.
  /// The configuration the places map and the viewer&#39;s location card need before they can draw a base layer: the tile source the administrator configured, and nothing else.  THE POLYGON: the value is a URL the BROWSER loads directly from the tile host -- the server never fetches, proxies or caches a tile, so a pan or a zoom never makes the photo server a tile CDN. What the tile host sees is the viewport, the zoom level and the browsing device&#39;s address; the pins the map draws come from this instance&#39;s own API and never cross the tile line. A deployment whose browser cannot reach a public tile host points the setting at a self-hosted or mirrored source; there is deliberately no bundled default (a public tile service&#39;s usage policy forbids carrying the traffic of an application&#39;s whole install base).  &#x60;tilejsonUrl&#x60; is null when no source is configured. That is not an error: the map renders its honest degraded state -- the pins stay, the base layer is stated as absent -- and the admin settings face is where it gets fixed. A stored value is handed back verbatim; its reachability is the browser&#39;s business, and a source that fails to load lands on the same stated absence. 
  ///
  /// Parameters:
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [MapConfigResponse] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<MapConfigResponse>> getMapConfig({ 
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/map/config';
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

    MapConfigResponse? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(MapConfigResponse),
      ) as MapConfigResponse;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<MapConfigResponse>(
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

  /// The caller&#39;s city grouping.
  /// The places surface: every place of the GeoNames reference data that holds at least one asset the caller can see -- the city grouping the places page renders as its grid. A place joins the list when it carries a caller-visible live asset (the album union of the caller&#39;s own sync and directory albums plus every album shared to them, and the timeline&#39;s live-row filters: not trashed, not hidden, no Live Photo component row); &#x60;assetCount&#x60; counts exactly those assets, so the same place can show different counts to different accounts -- the honest per-caller counting discipline the people surface pins, applied to cities.  &#x60;name&#x60; is the locale-joined display name: the &#x60;place_names&#x60; row of the requested locale when the place data carries one, the GeoNames canonical (English or romanized) name otherwise. Locale is a REQUEST parameter (the web app owns the active locale and passes it along); &#x60;en&#x60; is the default. A locale switch changes the join only -- the assets rows never carry locale strings, so switching languages never rewrites asset data.  &#x60;representativeCoverAssetIds&#x60; is the wall&#39;s derived-cover discipline applied to city groups: the place&#39;s (up to) four most recent caller-visible live members, taken_at descending with the asset id as the tiebreak -- the cover tiles the places page renders from. A listed place always carries at least one candidate (the listing itself proved a visible member exists).  The list is ordered by &#x60;assetCount&#x60; descending, the place id ascending as the deterministic tiebreak. It is not paginated: the same single-document shape the named-people listing uses, sized for a family instance&#39;s few thousand places.  Place resolution itself (which asset belongs to which city) is the ingestion pipeline&#39;s business -- the metadata job&#39;s GPS step and the admin re-resolution run -- and follows the place data baked into the server image; this endpoint only reads what those wrote. 
  ///
  /// Parameters:
  /// * [locale] - The display locale of the place names: `en` or `zh-Hans`, the instance's UI language set. Absent means `en`. A value outside the set answers 400 `VALIDATION_FAILED`. 
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [BuiltList<PlaceSummary>] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<BuiltList<PlaceSummary>>> listPlaces({ 
    String? locale = 'en',
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/places';
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
      if (locale != null) r'locale': encodeQueryParameter(_serializers, locale, const FullType(String)),
    };

    final _response = await _dio.request<Object>(
      _path,
      options: _options,
      queryParameters: _queryParameters,
      cancelToken: cancelToken,
      onSendProgress: onSendProgress,
      onReceiveProgress: onReceiveProgress,
    );

    BuiltList<PlaceSummary>? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(BuiltList, [FullType(PlaceSummary)]),
      ) as BuiltList<PlaceSummary>;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<BuiltList<PlaceSummary>>(
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
