//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'pairing_token_response.g.dart';

/// The one-time pairing token issued by `POST /auth/pairing/token`.
///
/// Properties:
/// * [token] - One-time pairing secret, 128 random bits as a URL-safe string. Render it as a QR code and never store or log it; the server keeps only its hash. 
/// * [expiresAt] - When the token expires (RFC 3339 UTC); five minutes after issuance. 
@BuiltValue()
abstract class PairingTokenResponse implements Built<PairingTokenResponse, PairingTokenResponseBuilder> {
  /// One-time pairing secret, 128 random bits as a URL-safe string. Render it as a QR code and never store or log it; the server keeps only its hash. 
  @BuiltValueField(wireName: r'token')
  String get token;

  /// When the token expires (RFC 3339 UTC); five minutes after issuance. 
  @BuiltValueField(wireName: r'expiresAt')
  DateTime get expiresAt;

  PairingTokenResponse._();

  factory PairingTokenResponse([void updates(PairingTokenResponseBuilder b)]) = _$PairingTokenResponse;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(PairingTokenResponseBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<PairingTokenResponse> get serializer => _$PairingTokenResponseSerializer();
}

class _$PairingTokenResponseSerializer implements PrimitiveSerializer<PairingTokenResponse> {
  @override
  final Iterable<Type> types = const [PairingTokenResponse, _$PairingTokenResponse];

  @override
  final String wireName = r'PairingTokenResponse';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    PairingTokenResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'token';
    yield serializers.serialize(
      object.token,
      specifiedType: const FullType(String),
    );
    yield r'expiresAt';
    yield serializers.serialize(
      object.expiresAt,
      specifiedType: const FullType(DateTime),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    PairingTokenResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required PairingTokenResponseBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'token':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.token = valueDes;
          break;
        case r'expiresAt':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(DateTime),
          ) as DateTime;
          result.expiresAt = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  PairingTokenResponse deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = PairingTokenResponseBuilder();
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


