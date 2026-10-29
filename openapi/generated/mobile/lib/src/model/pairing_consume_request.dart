//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'pairing_consume_request.g.dart';

/// Payload of `POST /auth/pairing/consume`.
///
/// Properties:
/// * [token] - The pairing token read from the QR code.
/// * [deviceName] - Human-readable device label shown in the session list, for example `Pixel 9`. 
/// * [deviceId] - Stable client-generated identifier of the physical device, for example a UUID the app persists; it lets the server recognize the same device across pairings. 
@BuiltValue()
abstract class PairingConsumeRequest implements Built<PairingConsumeRequest, PairingConsumeRequestBuilder> {
  /// The pairing token read from the QR code.
  @BuiltValueField(wireName: r'token')
  String get token;

  /// Human-readable device label shown in the session list, for example `Pixel 9`. 
  @BuiltValueField(wireName: r'deviceName')
  String get deviceName;

  /// Stable client-generated identifier of the physical device, for example a UUID the app persists; it lets the server recognize the same device across pairings. 
  @BuiltValueField(wireName: r'deviceId')
  String get deviceId;

  PairingConsumeRequest._();

  factory PairingConsumeRequest([void updates(PairingConsumeRequestBuilder b)]) = _$PairingConsumeRequest;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(PairingConsumeRequestBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<PairingConsumeRequest> get serializer => _$PairingConsumeRequestSerializer();
}

class _$PairingConsumeRequestSerializer implements PrimitiveSerializer<PairingConsumeRequest> {
  @override
  final Iterable<Type> types = const [PairingConsumeRequest, _$PairingConsumeRequest];

  @override
  final String wireName = r'PairingConsumeRequest';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    PairingConsumeRequest object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'token';
    yield serializers.serialize(
      object.token,
      specifiedType: const FullType(String),
    );
    yield r'deviceName';
    yield serializers.serialize(
      object.deviceName,
      specifiedType: const FullType(String),
    );
    yield r'deviceId';
    yield serializers.serialize(
      object.deviceId,
      specifiedType: const FullType(String),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    PairingConsumeRequest object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required PairingConsumeRequestBuilder result,
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
        case r'deviceName':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.deviceName = valueDes;
          break;
        case r'deviceId':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.deviceId = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  PairingConsumeRequest deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = PairingConsumeRequestBuilder();
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


