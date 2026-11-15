//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

import 'dart:async';

import 'package:built_value/json_object.dart';
import 'package:built_value/serializer.dart';
import 'package:dio/dio.dart';

import 'package:built_collection/built_collection.dart';
import 'package:lumemor_api/src/api_util.dart';
import 'package:lumemor_api/src/model/device_session.dart';
import 'package:lumemor_api/src/model/error.dart';
import 'package:lumemor_api/src/model/password_change_request.dart';
import 'package:lumemor_api/src/model/user_list_response.dart';
import 'package:lumemor_api/src/model/user_self_patch.dart';
import 'package:lumemor_api/src/model/user_self_response.dart';

class UsersApi {

  final Dio _dio;

  final Serializers _serializers;

  const UsersApi(this._dio, this._serializers);

  /// Change the caller&#39;s own password.
  /// The self-scoped password change behind the settings page&#39;s change-password section. The current password must verify against the account&#39;s stored Argon2id hash; the next one is stored as a fresh Argon2id hash (the same profile every creation path pins -- a rehash with a new salt, never a mutation of the old hash).  **The revocation contract, pinned:** a successful change revokes EVERY OTHER session of the account, across all devices -- web cookie sessions and mobile bearer sessions alike, their credentials failing authentication with 401 from the next request on -- while the session that performed the change STAYS signed in (the caller is not signed out of the device they just proved they own). The rationale is the credential-severance rule: a changed password means every other held credential of the account is stale, and a device that wants back in signs in with the new password.  A wrong current password answers 400 VALIDATION_FAILED naming &#x60;current&#x60; -- NOT 401, because the request&#39;&#39;s session is valid and healthy; only the request&#39;&#39;s own field failed to verify. (Clients must not treat it as a session expiry.) The next password follows the account rules everywhere: 8 to 128 runes. 
  ///
  /// Parameters:
  /// * [passwordChangeRequest] 
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future]
  /// Throws [DioException] if API call or serialization fails
  Future<Response<void>> changeUserPassword({ 
    required PasswordChangeRequest passwordChangeRequest,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/users/me/password';
    final _options = Options(
      method: r'PUT',
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
      const _type = FullType(PasswordChangeRequest);
      _bodyData = _serializers.serialize(passwordChangeRequest, specifiedType: _type);

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

    return _response;
  }

  /// Read the caller&#39;s own account.
  /// The self-scoped profile read behind the user settings page: the calling session&#39;s own account row -- the identity fields of the profile section, the appearance pair (theme and locale) the account persisted, and the download preferences the batch and whole-album download flows will consume. The endpoint is &#x60;me&#x60;-scoped by construction: the session that authenticates the request IS the identity, there is no userId parameter to get wrong, and one account can never read another&#39;s row through it.  The theme and the locale are the values the account chose on the settings page; a client that applies them (the web app&#39;s &#x60;.dark&#x60; class and i18n store) restores the account&#39;s appearance at sign-in. The download preferences are declarative account state: they persist here and every future download flow reads them as the account&#39;s default packaging (see &#x60;UserSelfPatch&#x60; for the honest consumption state). 
  ///
  /// Parameters:
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [UserSelfResponse] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<UserSelfResponse>> getUserSelf({ 
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/users/me';
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

    UserSelfResponse? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(UserSelfResponse),
      ) as UserSelfResponse;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<UserSelfResponse>(
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

  /// The instance&#39;s member list.
  /// The minimal member list of the instance: one entry per account, &#x60;email&#x60; ascending. Any authenticated account may enumerate it -- the family-instance posture: the accounts of a shared photo home are not secret from one another, and the album share dialog&#39;s pick list (the owner choosing whom to grant) is the list&#39;s first consumer.  The entries carry exactly the identity a pick list renders: &#x60;id&#x60;, &#x60;email&#x60;, &#x60;name&#x60;, &#x60;role&#x60; and &#x60;isDisabled&#x60; -- no timestamps, and a password hash never appears in any response. The display name leads there (the pick list says \&quot;Mum\&quot;, not her login address); &#x60;email&#x60; stays the fallback an empty name leaves and the disambiguator between two accounts that share a name. The disabled switch rides the entry so callers can present accounts that cannot sign in for what they are (granting to one still answers 400 through the share endpoint&#39;s own target rule).  The list is small by construction (a family instance holds a handful of accounts) and grows only through the admin surface&#39;s own create endpoint, so it answers every account in one response: there is no pagination, no filtering and no ordering parameter. This is NOT the admin management face (&#x60;GET /admin/users&#x60;): it is the minimal, non-managing pick list every authenticated account may read. 
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

  /// List the caller&#39;s signed-in devices.
  /// The device list behind the settings page&#39;s devices section: every active session of the calling account across devices, most recently used first, each entry carrying the device rendering&#39;s own fields -- and, unlike the auth surface&#39;s own &#x60;GET /auth/sessions&#x60;, the &#x60;isCurrent&#x60; marker derived by comparing the session id against the credential of THIS request, plus the session&#39;s creation-time IP address.  The list is the account&#39;s own sessions only (the &#x60;me&#x60; scoping again: the session IS the identity); a web session and a paired phone both appear. &#x60;ipAddress&#x60; is the address the session was created from (the login&#39;s or the pairing exchange&#39;s socket peer), omitted when the row predates the column or the address was not captured. It is an audit hint, not a geolocation claim. 
  ///
  /// Parameters:
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [BuiltList<DeviceSession>] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<BuiltList<DeviceSession>>> listUserDevices({ 
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/users/me/sessions';
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

    BuiltList<DeviceSession>? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(BuiltList, [FullType(DeviceSession)]),
      ) as BuiltList<DeviceSession>;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<BuiltList<DeviceSession>>(
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

  /// Revoke one of the caller&#39;s other devices.
  /// The device list&#39;s revoke button: signs out one device by deleting its session row. The endpoint is owner-scoped like the auth surface&#39;s own &#x60;DELETE /auth/sessions/{id}&#x60; -- an id that is unknown, belongs to another account, or is already revoked still answers the same 204, without revealing whether such a session exists.  THE ONE DIFFERENCE FROM THE AUTH SURFACE: the CURRENT session -- the one authenticating this very request, the row the device list marks &#x60;isCurrent&#x60; -- answers 400 VALIDATION_FAILED instead. The settings page&#39;s list is a remote-device manager; the current device&#39;s own exit is the sign-out flow (&#x60;POST /auth/logout&#x60;), and the 400 keeps a stray click on the wrong row from signing the caller out of the very session they are managing from. 
  ///
  /// Parameters:
  /// * [id] - Identifier of the session to revoke.
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future]
  /// Throws [DioException] if API call or serialization fails
  Future<Response<void>> revokeUserDevice({ 
    required String id,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/users/me/sessions/{id}'.replaceAll('{' r'id' '}', encodeQueryParameter(_serializers, id, const FullType(String)).toString());
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

  /// Patch the caller&#39;s own account.
  /// The self-scoped write behind the settings page&#39;s profile, appearance and download-preferences sections. Every field is optional: an absent field keeps its stored value, a present one replaces it, and the response is the complete saved account state (the caller adopts it as the fresh state, exactly like the storage settings save).  THE ME-SCOPING: the endpoint addresses the session&#39;s own account row only -- no userId parameter exists, and the session middleware&#39;s identity decides the row. The email is the one field whose uniqueness spans accounts: an address another account already owns answers 409 USER_EMAIL_TAKEN and nothing is written.  The theme and the locale persist for the NEXT sign-in and every client that reads them; the client performing the patch is expected to apply them immediately itself (the web app flips its &#x60;.dark&#x60; class and reloads its i18n store on the response, not on a re-login).  THE DOWNLOAD PREFERENCES split honestly in two. The embedded-motion switch is consumed today: the single-asset download reads it when it assembles a Live Photo&#39;s archive. The format, videos and sidecars switches stay declarative -- the batch and whole-album download endpoints do not exist yet (they arrive with a later surface), and those flows will read the account&#39;s default packaging when they arrive: a stored preference is never silently dropped, it is simply not consumed yet. 
  ///
  /// Parameters:
  /// * [userSelfPatch] 
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [UserSelfResponse] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<UserSelfResponse>> updateUserSelf({ 
    required UserSelfPatch userSelfPatch,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/users/me';
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
      const _type = FullType(UserSelfPatch);
      _bodyData = _serializers.serialize(userSelfPatch, specifiedType: _type);

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

    UserSelfResponse? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(UserSelfResponse),
      ) as UserSelfResponse;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<UserSelfResponse>(
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
