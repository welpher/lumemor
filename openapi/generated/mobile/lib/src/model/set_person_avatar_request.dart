//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'set_person_avatar_request.g.dart';

/// The face one person's avatar renders: one of the cluster's own faces, on an asset visible to the caller. 
///
/// Properties:
/// * [faceId] - Identifier of the face to render as the avatar.
@BuiltValue()
abstract class SetPersonAvatarRequest implements Built<SetPersonAvatarRequest, SetPersonAvatarRequestBuilder> {
  /// Identifier of the face to render as the avatar.
  @BuiltValueField(wireName: r'faceId')
  String get faceId;

  SetPersonAvatarRequest._();

  factory SetPersonAvatarRequest([void updates(SetPersonAvatarRequestBuilder b)]) = _$SetPersonAvatarRequest;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(SetPersonAvatarRequestBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<SetPersonAvatarRequest> get serializer => _$SetPersonAvatarRequestSerializer();
}

class _$SetPersonAvatarRequestSerializer implements PrimitiveSerializer<SetPersonAvatarRequest> {
  @override
  final Iterable<Type> types = const [SetPersonAvatarRequest, _$SetPersonAvatarRequest];

  @override
  final String wireName = r'SetPersonAvatarRequest';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    SetPersonAvatarRequest object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'faceId';
    yield serializers.serialize(
      object.faceId,
      specifiedType: const FullType(String),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    SetPersonAvatarRequest object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required SetPersonAvatarRequestBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'faceId':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.faceId = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  SetPersonAvatarRequest deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = SetPersonAvatarRequestBuilder();
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


