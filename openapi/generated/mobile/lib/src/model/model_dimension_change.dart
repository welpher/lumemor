//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'model_dimension_change.g.dart';

/// The dimension switch of a cross-dimension clip activation: what the embeddings column was and what it became. 
///
/// Properties:
/// * [from] - The embeddings column's dimension before the switch.
/// * [to] - The embeddings column's dimension after the switch.
@BuiltValue()
abstract class ModelDimensionChange implements Built<ModelDimensionChange, ModelDimensionChangeBuilder> {
  /// The embeddings column's dimension before the switch.
  @BuiltValueField(wireName: r'from')
  int get from;

  /// The embeddings column's dimension after the switch.
  @BuiltValueField(wireName: r'to')
  int get to;

  ModelDimensionChange._();

  factory ModelDimensionChange([void updates(ModelDimensionChangeBuilder b)]) = _$ModelDimensionChange;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(ModelDimensionChangeBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<ModelDimensionChange> get serializer => _$ModelDimensionChangeSerializer();
}

class _$ModelDimensionChangeSerializer implements PrimitiveSerializer<ModelDimensionChange> {
  @override
  final Iterable<Type> types = const [ModelDimensionChange, _$ModelDimensionChange];

  @override
  final String wireName = r'ModelDimensionChange';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    ModelDimensionChange object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'from';
    yield serializers.serialize(
      object.from,
      specifiedType: const FullType(int),
    );
    yield r'to';
    yield serializers.serialize(
      object.to,
      specifiedType: const FullType(int),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    ModelDimensionChange object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required ModelDimensionChangeBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'from':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(int),
          ) as int;
          result.from = valueDes;
          break;
        case r'to':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(int),
          ) as int;
          result.to = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  ModelDimensionChange deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = ModelDimensionChangeBuilder();
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


