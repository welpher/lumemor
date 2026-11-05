//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

import 'dart:async';

import 'package:built_value/json_object.dart';
import 'package:built_value/serializer.dart';
import 'package:dio/dio.dart';

import 'package:lumemor_api/src/model/error.dart';
import 'package:lumemor_api/src/model/user_list_response.dart';

class UsersApi {

  final Dio _dio;

  final Serializers _serializers;

  const UsersApi(this._dio, this._serializers);

  /// The instance&#39;s member list.
  /// The minimal member list of the instance: one entry per account, &#x60;email&#x60; ascending. Any authenticated account may enumerate it -- the family-instance posture: the accounts of a shared photo home are not secret from one another, and the album share dialog&#39;s pick list (the owner choosing whom to grant) is the list&#39;s first consumer.  The entries carry exactly the identity a pick list renders: &#x60;id&#x60;, &#x60;email&#x60;, &#x60;role&#x60; and &#x60;isDisabled&#x60; -- no display name, no timestamps, and a password hash never appears in any response. The disabled switch rides the entry so callers can present accounts that cannot sign in for what they are (granting to one still answers 400 through the share endpoint&#39;s own target rule).  The list is small by construction (a family instance holds a handful of accounts) and grows only through the admin surface&#39;s own create endpoint, so it answers every account in one response: there is no pagination, no filtering and no ordering parameter. This is NOT the admin management face (&#x60;GET /admin/users&#x60;): it is the minimal, non-managing pick list every authenticated account may read. 
  ///
  /// Parameters:
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [UserListResponse] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<UserListResponse>> listInstanceUsers({ 
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/users';
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

    UserListResponse? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(UserListResponse),
      ) as UserListResponse;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<UserListResponse>(
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
