//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:lumemor_api/src/model/session_info.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'bearer_session.g.dart';

/// The mobile session minted by consuming a pairing token: the bearer credential plus the session it authenticates. 
///
/// Properties:
/// * [token] - Bearer credential of the new mobile session; send it as `Authorization: Bearer <token>`. It appears only in this response, and the server stores only its hash: a device that loses it must pair again. 
/// * [session] 
@BuiltValue()
abstract class BearerSession implements Built<BearerSession, BearerSessionBuilder> {
  /// Bearer credential of the new mobile session; send it as `Authorization: Bearer <token>`. It appears only in this response, and the server stores only its hash: a device that loses it must pair again. 
  @BuiltValueField(wireName: r'token')
  String get token;

  @BuiltValueField(wireName: r'session')
  SessionInfo get session;

  BearerSession._();

  factory BearerSession([void updates(BearerSessionBuilder b)]) = _$BearerSession;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(BearerSessionBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<BearerSession> get serializer => _$BearerSessionSerializer();
}

class _$BearerSessionSerializer implements PrimitiveSerializer<BearerSession> {
  @override
  final Iterable<Type> types = const [BearerSession, _$BearerSession];

  @override
  final String wireName = r'BearerSession';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    BearerSession object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'token';
    yield serializers.serialize(
      object.token,
      specifiedType: const FullType(String),
    );
    yield r'session';
    yield serializers.serialize(
      object.session,
      specifiedType: const FullType(SessionInfo),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    BearerSession object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required BearerSessionBuilder result,
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
        case r'session':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(SessionInfo),
          ) as SessionInfo;
          result.session.replace(valueDes);
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  BearerSession deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = BearerSessionBuilder();
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


