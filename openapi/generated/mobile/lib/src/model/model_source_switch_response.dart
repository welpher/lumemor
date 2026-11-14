//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:lumemor_api/src/model/model_ref.dart';
import 'package:lumemor_api/src/model/model_source_switch_response_dimension_changed.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'model_source_switch_response.g.dart';

/// What one switch did: the slot's newly active model, the destructive arm's facts when the dimension changed, and the recalculation run's id when the switch's semantics enqueue one (the face slots and a cross-dimension embed switch; a same-dimension embed switch keeps both generations and recomputes nothing). 
///
/// Properties:
/// * [activated] 
/// * [dimensionChanged] 
/// * [wipedEmbeddings] - The number of embeddings the dimension switch deleted.
/// * [recalcRunId] - The enqueued recalculation run's identifier, null when none is due.
@BuiltValue()
abstract class ModelSourceSwitchResponse implements Built<ModelSourceSwitchResponse, ModelSourceSwitchResponseBuilder> {
  @BuiltValueField(wireName: r'activated')
  ModelRef get activated;

  @BuiltValueField(wireName: r'dimensionChanged')
  ModelSourceSwitchResponseDimensionChanged? get dimensionChanged;

  /// The number of embeddings the dimension switch deleted.
  @BuiltValueField(wireName: r'wipedEmbeddings')
  int? get wipedEmbeddings;

  /// The enqueued recalculation run's identifier, null when none is due.
  @BuiltValueField(wireName: r'recalcRunId')
  String? get recalcRunId;

  ModelSourceSwitchResponse._();

  factory ModelSourceSwitchResponse([void updates(ModelSourceSwitchResponseBuilder b)]) = _$ModelSourceSwitchResponse;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(ModelSourceSwitchResponseBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<ModelSourceSwitchResponse> get serializer => _$ModelSourceSwitchResponseSerializer();
}

class _$ModelSourceSwitchResponseSerializer implements PrimitiveSerializer<ModelSourceSwitchResponse> {
  @override
  final Iterable<Type> types = const [ModelSourceSwitchResponse, _$ModelSourceSwitchResponse];

  @override
  final String wireName = r'ModelSourceSwitchResponse';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    ModelSourceSwitchResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'activated';
    yield serializers.serialize(
      object.activated,
      specifiedType: const FullType(ModelRef),
    );
    yield r'dimensionChanged';
    yield object.dimensionChanged == null ? null : serializers.serialize(
      object.dimensionChanged,
      specifiedType: const FullType.nullable(ModelSourceSwitchResponseDimensionChanged),
    );
    yield r'wipedEmbeddings';
    yield object.wipedEmbeddings == null ? null : serializers.serialize(
      object.wipedEmbeddings,
      specifiedType: const FullType.nullable(int),
    );
    yield r'recalcRunId';
    yield object.recalcRunId == null ? null : serializers.serialize(
      object.recalcRunId,
      specifiedType: const FullType.nullable(String),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    ModelSourceSwitchResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required ModelSourceSwitchResponseBuilder result,
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
            specifiedType: const FullType.nullable(ModelSourceSwitchResponseDimensionChanged),
          ) as ModelSourceSwitchResponseDimensionChanged?;
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
        case r'recalcRunId':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(String),
          ) as String?;
          if (valueDes == null) continue;
          result.recalcRunId = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  ModelSourceSwitchResponse deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = ModelSourceSwitchResponseBuilder();
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


