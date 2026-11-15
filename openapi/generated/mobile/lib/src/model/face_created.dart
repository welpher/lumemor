//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'face_created.g.dart';

/// One committed manual face: its identity and the crop URL. The face carries no cluster yet -- the assign flow (the move or the from-face creation) is the caller's next step, and that is where the drawn face becomes someone. 
///
/// Properties:
/// * [id] - Identifier of the new face row.
/// * [thumbnail] - URL of the new face's crop: the face crop endpoint with the new face's id (the same shape the detail's faces carry). 
@BuiltValue()
abstract class FaceCreated implements Built<FaceCreated, FaceCreatedBuilder> {
  /// Identifier of the new face row.
  @BuiltValueField(wireName: r'id')
  String get id;

  /// URL of the new face's crop: the face crop endpoint with the new face's id (the same shape the detail's faces carry). 
  @BuiltValueField(wireName: r'thumbnail')
  String get thumbnail;

  FaceCreated._();

  factory FaceCreated([void updates(FaceCreatedBuilder b)]) = _$FaceCreated;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(FaceCreatedBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<FaceCreated> get serializer => _$FaceCreatedSerializer();
}

class _$FaceCreatedSerializer implements PrimitiveSerializer<FaceCreated> {
  @override
  final Iterable<Type> types = const [FaceCreated, _$FaceCreated];

  @override
  final String wireName = r'FaceCreated';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    FaceCreated object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'id';
    yield serializers.serialize(
      object.id,
      specifiedType: const FullType(String),
    );
    yield r'thumbnail';
    yield serializers.serialize(
      object.thumbnail,
      specifiedType: const FullType(String),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    FaceCreated object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required FaceCreatedBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'id':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.id = valueDes;
          break;
        case r'thumbnail':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.thumbnail = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  FaceCreated deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = FaceCreatedBuilder();
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


