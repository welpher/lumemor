//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'password_change_request.g.dart';

/// Payload of `PUT /users/me/password`.
///
/// Properties:
/// * [current] - The account's current password; it must verify against the stored Argon2id hash before anything is written. 
/// * [next] - The new password; 8 to 128 runes, stored as a fresh Argon2id hash.
@BuiltValue()
abstract class PasswordChangeRequest implements Built<PasswordChangeRequest, PasswordChangeRequestBuilder> {
  /// The account's current password; it must verify against the stored Argon2id hash before anything is written. 
  @BuiltValueField(wireName: r'current')
  String get current;

  /// The new password; 8 to 128 runes, stored as a fresh Argon2id hash.
  @BuiltValueField(wireName: r'next')
  String get next;

  PasswordChangeRequest._();

  factory PasswordChangeRequest([void updates(PasswordChangeRequestBuilder b)]) = _$PasswordChangeRequest;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(PasswordChangeRequestBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<PasswordChangeRequest> get serializer => _$PasswordChangeRequestSerializer();
}

class _$PasswordChangeRequestSerializer implements PrimitiveSerializer<PasswordChangeRequest> {
  @override
  final Iterable<Type> types = const [PasswordChangeRequest, _$PasswordChangeRequest];

  @override
  final String wireName = r'PasswordChangeRequest';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    PasswordChangeRequest object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'current';
    yield serializers.serialize(
      object.current,
      specifiedType: const FullType(String),
    );
    yield r'next';
    yield serializers.serialize(
      object.next,
      specifiedType: const FullType(String),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    PasswordChangeRequest object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required PasswordChangeRequestBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'current':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.current = valueDes;
          break;
        case r'next':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.next = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  PasswordChangeRequest deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = PasswordChangeRequestBuilder();
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


