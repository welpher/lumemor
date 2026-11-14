//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'model_source_switch_response_dimension_changed.g.dart';

/// Set when the switch re-typed the vector column.
///
/// Properties:
/// * [from] 
/// * [to] 
@BuiltValue()
abstract class ModelSourceSwitchResponseDimensionChanged implements Built<ModelSourceSwitchResponseDimensionChanged, ModelSourceSwitchResponseDimensionChangedBuilder> {
  @BuiltValueField(wireName: r'from')
  int get from;

  @BuiltValueField(wireName: r'to')
  int get to;

  ModelSourceSwitchResponseDimensionChanged._();

  factory ModelSourceSwitchResponseDimensionChanged([void updates(ModelSourceSwitchResponseDimensionChangedBuilder b)]) = _$ModelSourceSwitchResponseDimensionChanged;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(ModelSourceSwitchResponseDimensionChangedBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<ModelSourceSwitchResponseDimensionChanged> get serializer => _$ModelSourceSwitchResponseDimensionChangedSerializer();
}

class _$ModelSourceSwitchResponseDimensionChangedSerializer implements PrimitiveSerializer<ModelSourceSwitchResponseDimensionChanged> {
  @override
  final Iterable<Type> types = const [ModelSourceSwitchResponseDimensionChanged, _$ModelSourceSwitchResponseDimensionChanged];

  @override
  final String wireName = r'ModelSourceSwitchResponseDimensionChanged';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    ModelSourceSwitchResponseDimensionChanged object, {
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
    ModelSourceSwitchResponseDimensionChanged object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required ModelSourceSwitchResponseDimensionChangedBuilder result,
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
  ModelSourceSwitchResponseDimensionChanged deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = ModelSourceSwitchResponseDimensionChangedBuilder();
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


