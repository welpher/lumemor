//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:lumemor_api/src/model/model_slot.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'model_ref.g.dart';

/// The identity of one model: its slug id and its slot. 
///
/// Properties:
/// * [id] - The model's stable slug identifier.
/// * [slot] 
@BuiltValue()
abstract class ModelRef implements Built<ModelRef, ModelRefBuilder> {
  /// The model's stable slug identifier.
  @BuiltValueField(wireName: r'id')
  String get id;

  @BuiltValueField(wireName: r'slot')
  ModelSlot get slot;
  // enum slotEnum {  face_detect,  face_recognize,  clip_embed,  clip_text,  ocr,  };

  ModelRef._();

  factory ModelRef([void updates(ModelRefBuilder b)]) = _$ModelRef;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(ModelRefBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<ModelRef> get serializer => _$ModelRefSerializer();
}

class _$ModelRefSerializer implements PrimitiveSerializer<ModelRef> {
  @override
  final Iterable<Type> types = const [ModelRef, _$ModelRef];

  @override
  final String wireName = r'ModelRef';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    ModelRef object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'id';
    yield serializers.serialize(
      object.id,
      specifiedType: const FullType(String),
    );
    yield r'slot';
    yield serializers.serialize(
      object.slot,
      specifiedType: const FullType(ModelSlot),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    ModelRef object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required ModelRefBuilder result,
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
        case r'slot':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(ModelSlot),
          ) as ModelSlot;
          result.slot = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  ModelRef deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = ModelRefBuilder();
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


