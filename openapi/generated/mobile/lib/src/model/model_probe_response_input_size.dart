//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'model_probe_response_input_size.g.dart';

/// The model's input geometry, read off the graph.
///
/// Properties:
/// * [width] 
/// * [height] 
@BuiltValue()
abstract class ModelProbeResponseInputSize implements Built<ModelProbeResponseInputSize, ModelProbeResponseInputSizeBuilder> {
  @BuiltValueField(wireName: r'width')
  int get width;

  @BuiltValueField(wireName: r'height')
  int get height;

  ModelProbeResponseInputSize._();

  factory ModelProbeResponseInputSize([void updates(ModelProbeResponseInputSizeBuilder b)]) = _$ModelProbeResponseInputSize;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(ModelProbeResponseInputSizeBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<ModelProbeResponseInputSize> get serializer => _$ModelProbeResponseInputSizeSerializer();
}

class _$ModelProbeResponseInputSizeSerializer implements PrimitiveSerializer<ModelProbeResponseInputSize> {
  @override
  final Iterable<Type> types = const [ModelProbeResponseInputSize, _$ModelProbeResponseInputSize];

  @override
  final String wireName = r'ModelProbeResponseInputSize';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    ModelProbeResponseInputSize object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'width';
    yield serializers.serialize(
      object.width,
      specifiedType: const FullType(int),
    );
    yield r'height';
    yield serializers.serialize(
      object.height,
      specifiedType: const FullType(int),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    ModelProbeResponseInputSize object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required ModelProbeResponseInputSizeBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'width':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(int),
          ) as int;
          result.width = valueDes;
          break;
        case r'height':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(int),
          ) as int;
          result.height = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  ModelProbeResponseInputSize deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = ModelProbeResponseInputSizeBuilder();
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


