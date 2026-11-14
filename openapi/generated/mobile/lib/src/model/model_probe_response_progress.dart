//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'model_probe_response_progress.g.dart';

/// The URL download's `{done, total}` counts while running, null otherwise.
///
/// Properties:
/// * [done] 
/// * [total] 
@BuiltValue()
abstract class ModelProbeResponseProgress implements Built<ModelProbeResponseProgress, ModelProbeResponseProgressBuilder> {
  @BuiltValueField(wireName: r'done')
  int get done;

  @BuiltValueField(wireName: r'total')
  int get total;

  ModelProbeResponseProgress._();

  factory ModelProbeResponseProgress([void updates(ModelProbeResponseProgressBuilder b)]) = _$ModelProbeResponseProgress;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(ModelProbeResponseProgressBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<ModelProbeResponseProgress> get serializer => _$ModelProbeResponseProgressSerializer();
}

class _$ModelProbeResponseProgressSerializer implements PrimitiveSerializer<ModelProbeResponseProgress> {
  @override
  final Iterable<Type> types = const [ModelProbeResponseProgress, _$ModelProbeResponseProgress];

  @override
  final String wireName = r'ModelProbeResponseProgress';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    ModelProbeResponseProgress object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'done';
    yield serializers.serialize(
      object.done,
      specifiedType: const FullType(int),
    );
    yield r'total';
    yield serializers.serialize(
      object.total,
      specifiedType: const FullType(int),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    ModelProbeResponseProgress object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required ModelProbeResponseProgressBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'done':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(int),
          ) as int;
          result.done = valueDes;
          break;
        case r'total':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(int),
          ) as int;
          result.total = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  ModelProbeResponseProgress deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = ModelProbeResponseProgressBuilder();
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


