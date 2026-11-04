//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'move_face_response.g.dart';

/// One committed face move.
///
/// Properties:
/// * [faceId] - Identifier of the moved face.
/// * [clusterId] - The cluster the face occupies after the move.
@BuiltValue()
abstract class MoveFaceResponse implements Built<MoveFaceResponse, MoveFaceResponseBuilder> {
  /// Identifier of the moved face.
  @BuiltValueField(wireName: r'faceId')
  String get faceId;

  /// The cluster the face occupies after the move.
  @BuiltValueField(wireName: r'clusterId')
  String get clusterId;

  MoveFaceResponse._();

  factory MoveFaceResponse([void updates(MoveFaceResponseBuilder b)]) = _$MoveFaceResponse;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(MoveFaceResponseBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<MoveFaceResponse> get serializer => _$MoveFaceResponseSerializer();
}

class _$MoveFaceResponseSerializer implements PrimitiveSerializer<MoveFaceResponse> {
  @override
  final Iterable<Type> types = const [MoveFaceResponse, _$MoveFaceResponse];

  @override
  final String wireName = r'MoveFaceResponse';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    MoveFaceResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'faceId';
    yield serializers.serialize(
      object.faceId,
      specifiedType: const FullType(String),
    );
    yield r'clusterId';
    yield serializers.serialize(
      object.clusterId,
      specifiedType: const FullType(String),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    MoveFaceResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required MoveFaceResponseBuilder result,
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
        case r'clusterId':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.clusterId = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  MoveFaceResponse deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = MoveFaceResponseBuilder();
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


