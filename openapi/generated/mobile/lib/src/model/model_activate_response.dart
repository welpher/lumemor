//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:lumemor_api/src/model/model_ref.dart';
import 'package:lumemor_api/src/model/model_dimension_change.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'model_activate_response.g.dart';

/// The report of one activation. The optional fields appear exactly on a cross-dimension switch of the clip-embed slot -- the destructive arm -- so their absence IS the same-dimension guarantee that nothing was deleted. 
///
/// Properties:
/// * [activated] 
/// * [dimensionChanged] 
/// * [wipedEmbeddings] - The number of clip embedding rows the cross-dimension switch deleted inside its transaction. 
@BuiltValue()
abstract class ModelActivateResponse implements Built<ModelActivateResponse, ModelActivateResponseBuilder> {
  @BuiltValueField(wireName: r'activated')
  ModelRef get activated;

  @BuiltValueField(wireName: r'dimensionChanged')
  ModelDimensionChange? get dimensionChanged;

  /// The number of clip embedding rows the cross-dimension switch deleted inside its transaction. 
  @BuiltValueField(wireName: r'wipedEmbeddings')
  int? get wipedEmbeddings;

  ModelActivateResponse._();

  factory ModelActivateResponse([void updates(ModelActivateResponseBuilder b)]) = _$ModelActivateResponse;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(ModelActivateResponseBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<ModelActivateResponse> get serializer => _$ModelActivateResponseSerializer();
}

class _$ModelActivateResponseSerializer implements PrimitiveSerializer<ModelActivateResponse> {
  @override
  final Iterable<Type> types = const [ModelActivateResponse, _$ModelActivateResponse];

  @override
  final String wireName = r'ModelActivateResponse';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    ModelActivateResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'activated';
    yield serializers.serialize(
      object.activated,
      specifiedType: const FullType(ModelRef),
    );
    if (object.dimensionChanged != null) {
      yield r'dimensionChanged';
      yield serializers.serialize(
        object.dimensionChanged,
        specifiedType: const FullType(ModelDimensionChange),
      );
    }
    if (object.wipedEmbeddings != null) {
      yield r'wipedEmbeddings';
      yield serializers.serialize(
        object.wipedEmbeddings,
        specifiedType: const FullType(int),
      );
    }
  }

  @override
  Object serialize(
    Serializers serializers,
    ModelActivateResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required ModelActivateResponseBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'activated':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(ModelRef),
          ) as ModelRef;
          result.activated.replace(valueDes);
          break;
        case r'dimensionChanged':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(ModelDimensionChange),
          ) as ModelDimensionChange?;
          if (valueDes == null) continue;
          result.dimensionChanged.replace(valueDes);
          break;
        case r'wipedEmbeddings':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(int),
          ) as int?;
          if (valueDes == null) continue;
          result.wipedEmbeddings = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  ModelActivateResponse deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = ModelActivateResponseBuilder();
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


