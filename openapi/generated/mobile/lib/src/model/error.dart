//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_collection/built_collection.dart';
import 'package:built_value/json_object.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'error.g.dart';

/// Error envelope shared by all endpoints. `code` is the authoritative registry of stable error codes; clients must not parse `message`. 
///
/// Properties:
/// * [code] - Stable error code. The i18n translation key is the lowercase form of this value. 
/// * [message] - English technical description for debugging.
/// * [details] - Optional structured context, for example the offending field names of a `VALIDATION_FAILED` request. Its shape is code-specific. 
@BuiltValue()
abstract class Error implements Built<Error, ErrorBuilder> {
  /// Stable error code. The i18n translation key is the lowercase form of this value. 
  @BuiltValueField(wireName: r'code')
  ErrorCodeEnum get code;
  // enum codeEnum {  SETUP_ALREADY_COMPLETED,  VALIDATION_FAILED,  INTERNAL,  AUTH_FORBIDDEN,  AUTH_INVALID_CREDENTIALS,  AUTH_RATE_LIMITED,  PAIRING_TOKEN_INVALID,  ASSET_TOO_LARGE,  ASSET_NOT_FOUND,  ASSET_NOT_READY,  USER_EMAIL_TAKEN,  USER_NOT_FOUND,  MODEL_DOWNLOAD_FAILED,  };

  /// English technical description for debugging.
  @BuiltValueField(wireName: r'message')
  String? get message;

  /// Optional structured context, for example the offending field names of a `VALIDATION_FAILED` request. Its shape is code-specific. 
  @BuiltValueField(wireName: r'details')
  BuiltMap<String, JsonObject?>? get details;

  Error._();

  factory Error([void updates(ErrorBuilder b)]) = _$Error;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(ErrorBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<Error> get serializer => _$ErrorSerializer();
}

class _$ErrorSerializer implements PrimitiveSerializer<Error> {
  @override
  final Iterable<Type> types = const [Error, _$Error];

  @override
  final String wireName = r'Error';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    Error object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'code';
    yield serializers.serialize(
      object.code,
      specifiedType: const FullType(ErrorCodeEnum),
    );
    if (object.message != null) {
      yield r'message';
      yield serializers.serialize(
        object.message,
        specifiedType: const FullType(String),
      );
    }
    if (object.details != null) {
      yield r'details';
      yield serializers.serialize(
        object.details,
        specifiedType: const FullType(BuiltMap, [FullType(String), FullType.nullable(JsonObject)]),
      );
    }
  }

  @override
  Object serialize(
    Serializers serializers,
    Error object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required ErrorBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'code':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(ErrorCodeEnum),
          ) as ErrorCodeEnum;
          result.code = valueDes;
          break;
        case r'message':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(String),
          ) as String?;
          if (valueDes == null) continue;
          result.message = valueDes;
          break;
        case r'details':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(BuiltMap, [FullType(String), FullType.nullable(JsonObject)]),
          ) as BuiltMap<String, JsonObject?>?;
          if (valueDes == null) continue;
          result.details.replace(valueDes);
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  Error deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = ErrorBuilder();
    final serializedList = (serialized as Iterable<Object?>).toList();
    final unhandled = <Object?>[];
    _deserializeProperties(
      serializers,
      serialized,
      specifiedType: specifiedType,
      serializedList: serializedList,
      unhandled: unhandled,
      result: result,
    );
    return result.build();
  }
}


/// Stable error code. The i18n translation key is the lowercase form of this value. 
class ErrorCodeEnum extends EnumClass {

  @BuiltValueEnumConst(wireName: r'SETUP_ALREADY_COMPLETED')
  static const ErrorCodeEnum SETUP_ALREADY_COMPLETED = _$errorCodeEnum_SETUP_ALREADY_COMPLETED;
  @BuiltValueEnumConst(wireName: r'VALIDATION_FAILED')
  static const ErrorCodeEnum VALIDATION_FAILED = _$errorCodeEnum_VALIDATION_FAILED;
  @BuiltValueEnumConst(wireName: r'INTERNAL')
  static const ErrorCodeEnum INTERNAL = _$errorCodeEnum_INTERNAL;
  @BuiltValueEnumConst(wireName: r'AUTH_FORBIDDEN')
  static const ErrorCodeEnum AUTH_FORBIDDEN = _$errorCodeEnum_AUTH_FORBIDDEN;
  @BuiltValueEnumConst(wireName: r'AUTH_INVALID_CREDENTIALS')
  static const ErrorCodeEnum AUTH_INVALID_CREDENTIALS = _$errorCodeEnum_AUTH_INVALID_CREDENTIALS;
  @BuiltValueEnumConst(wireName: r'AUTH_RATE_LIMITED')
  static const ErrorCodeEnum AUTH_RATE_LIMITED = _$errorCodeEnum_AUTH_RATE_LIMITED;
  @BuiltValueEnumConst(wireName: r'PAIRING_TOKEN_INVALID')
  static const ErrorCodeEnum PAIRING_TOKEN_INVALID = _$errorCodeEnum_PAIRING_TOKEN_INVALID;
  @BuiltValueEnumConst(wireName: r'ASSET_TOO_LARGE')
  static const ErrorCodeEnum ASSET_TOO_LARGE = _$errorCodeEnum_ASSET_TOO_LARGE;
  @BuiltValueEnumConst(wireName: r'ASSET_NOT_FOUND')
  static const ErrorCodeEnum ASSET_NOT_FOUND = _$errorCodeEnum_ASSET_NOT_FOUND;
  @BuiltValueEnumConst(wireName: r'ASSET_NOT_READY')
  static const ErrorCodeEnum ASSET_NOT_READY = _$errorCodeEnum_ASSET_NOT_READY;
  @BuiltValueEnumConst(wireName: r'USER_EMAIL_TAKEN')
  static const ErrorCodeEnum USER_EMAIL_TAKEN = _$errorCodeEnum_USER_EMAIL_TAKEN;
  @BuiltValueEnumConst(wireName: r'USER_NOT_FOUND')
  static const ErrorCodeEnum USER_NOT_FOUND = _$errorCodeEnum_USER_NOT_FOUND;
  @BuiltValueEnumConst(wireName: r'MODEL_DOWNLOAD_FAILED')
  static const ErrorCodeEnum MODEL_DOWNLOAD_FAILED = _$errorCodeEnum_MODEL_DOWNLOAD_FAILED;

  static Serializer<ErrorCodeEnum> get serializer => _$errorCodeEnumSerializer;

  const ErrorCodeEnum._(String name): super(name);

  static BuiltSet<ErrorCodeEnum> get values => _$errorCodeEnumValues;
  static ErrorCodeEnum valueOf(String name) => _$errorCodeEnumValueOf(name);
}

