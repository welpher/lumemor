//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'model_response_input_size.g.dart';

/// The model's input geometry. Null when the model's manifest is unreadable. 
///
/// Properties:
/// * [width] - Input width in pixels.
/// * [height] - Input height in pixels.
@BuiltValue()
abstract class ModelResponseInputSize implements Built<ModelResponseInputSize, ModelResponseInputSizeBuilder> {
  /// Input width in pixels.
  @BuiltValueField(wireName: r'width')
  int get width;

  /// Input height in pixels.
  @BuiltValueField(wireName: r'height')
  int get height;

  ModelResponseInputSize._();

  factory ModelResponseInputSize([void updates(ModelResponseInputSizeBuilder b)]) = _$ModelResponseInputSize;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(ModelResponseInputSizeBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<ModelResponseInputSize> get serializer => _$ModelResponseInputSizeSerializer();
}

class _$ModelResponseInputSizeSerializer implements PrimitiveSerializer<ModelResponseInputSize> {
  @override
  final Iterable<Type> types = const [ModelResponseInputSize, _$ModelResponseInputSize];

  @override
  final String wireName = r'ModelResponseInputSize';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    ModelResponseInputSize object, {
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
    ModelResponseInputSize object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required ModelResponseInputSizeBuilder result,
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
  ModelResponseInputSize deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = ModelResponseInputSizeBuilder();
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


