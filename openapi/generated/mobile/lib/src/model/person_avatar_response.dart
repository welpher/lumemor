//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'person_avatar_response.g.dart';

/// One person's avatar face after the write.
///
/// Properties:
/// * [clusterId] - Identifier of the face cluster.
/// * [avatarFaceId] - The stored avatar face.
@BuiltValue()
abstract class PersonAvatarResponse implements Built<PersonAvatarResponse, PersonAvatarResponseBuilder> {
  /// Identifier of the face cluster.
  @BuiltValueField(wireName: r'clusterId')
  String get clusterId;

  /// The stored avatar face.
  @BuiltValueField(wireName: r'avatarFaceId')
  String get avatarFaceId;

  PersonAvatarResponse._();

  factory PersonAvatarResponse([void updates(PersonAvatarResponseBuilder b)]) = _$PersonAvatarResponse;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(PersonAvatarResponseBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<PersonAvatarResponse> get serializer => _$PersonAvatarResponseSerializer();
}

class _$PersonAvatarResponseSerializer implements PrimitiveSerializer<PersonAvatarResponse> {
  @override
  final Iterable<Type> types = const [PersonAvatarResponse, _$PersonAvatarResponse];

  @override
  final String wireName = r'PersonAvatarResponse';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    PersonAvatarResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'clusterId';
    yield serializers.serialize(
      object.clusterId,
      specifiedType: const FullType(String),
    );
    yield r'avatarFaceId';
    yield serializers.serialize(
      object.avatarFaceId,
      specifiedType: const FullType(String),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    PersonAvatarResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required PersonAvatarResponseBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'clusterId':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.clusterId = valueDes;
          break;
        case r'avatarFaceId':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.avatarFaceId = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  PersonAvatarResponse deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = PersonAvatarResponseBuilder();
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


