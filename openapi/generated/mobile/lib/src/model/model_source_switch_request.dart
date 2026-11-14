//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_collection/built_collection.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'model_source_switch_request.g.dart';

/// The source a slot is switched onto. `kind` selects the shape: `builtin` takes `modelId` (an official-list entry, no probe needed), `url` and `file` take a `probeId` whose checks finished OK. `confirmWipe` is required when the switch would change the embed slot's vector dimension. 
///
/// Properties:
/// * [kind] 
/// * [modelId] - The official-list entry, for kind `builtin`.
/// * [probeId] - The finished probe, for kinds `url` and `file`.
/// * [confirmWipe] - The explicit confirmation a cross-dimension embed switch requires (the wipe cannot be undone). 
@BuiltValue()
abstract class ModelSourceSwitchRequest implements Built<ModelSourceSwitchRequest, ModelSourceSwitchRequestBuilder> {
  @BuiltValueField(wireName: r'kind')
  ModelSourceSwitchRequestKindEnum get kind;
  // enum kindEnum {  builtin,  url,  file,  };

  /// The official-list entry, for kind `builtin`.
  @BuiltValueField(wireName: r'modelId')
  String? get modelId;

  /// The finished probe, for kinds `url` and `file`.
  @BuiltValueField(wireName: r'probeId')
  String? get probeId;

  /// The explicit confirmation a cross-dimension embed switch requires (the wipe cannot be undone). 
  @BuiltValueField(wireName: r'confirmWipe')
  bool? get confirmWipe;

  ModelSourceSwitchRequest._();

  factory ModelSourceSwitchRequest([void updates(ModelSourceSwitchRequestBuilder b)]) = _$ModelSourceSwitchRequest;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(ModelSourceSwitchRequestBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<ModelSourceSwitchRequest> get serializer => _$ModelSourceSwitchRequestSerializer();
}

class _$ModelSourceSwitchRequestSerializer implements PrimitiveSerializer<ModelSourceSwitchRequest> {
  @override
  final Iterable<Type> types = const [ModelSourceSwitchRequest, _$ModelSourceSwitchRequest];

  @override
  final String wireName = r'ModelSourceSwitchRequest';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    ModelSourceSwitchRequest object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'kind';
    yield serializers.serialize(
      object.kind,
      specifiedType: const FullType(ModelSourceSwitchRequestKindEnum),
    );
    if (object.modelId != null) {
      yield r'modelId';
      yield serializers.serialize(
        object.modelId,
        specifiedType: const FullType(String),
      );
    }
    if (object.probeId != null) {
      yield r'probeId';
      yield serializers.serialize(
        object.probeId,
        specifiedType: const FullType(String),
      );
    }
    if (object.confirmWipe != null) {
      yield r'confirmWipe';
      yield serializers.serialize(
        object.confirmWipe,
        specifiedType: const FullType(bool),
      );
    }
  }

  @override
  Object serialize(
    Serializers serializers,
    ModelSourceSwitchRequest object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required ModelSourceSwitchRequestBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'kind':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(ModelSourceSwitchRequestKindEnum),
          ) as ModelSourceSwitchRequestKindEnum;
          result.kind = valueDes;
          break;
        case r'modelId':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(String),
          ) as String?;
          if (valueDes == null) continue;
          result.modelId = valueDes;
          break;
        case r'probeId':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(String),
          ) as String?;
          if (valueDes == null) continue;
          result.probeId = valueDes;
          break;
        case r'confirmWipe':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(bool),
          ) as bool?;
          if (valueDes == null) continue;
          result.confirmWipe = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  ModelSourceSwitchRequest deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = ModelSourceSwitchRequestBuilder();
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


class ModelSourceSwitchRequestKindEnum extends EnumClass {

  @BuiltValueEnumConst(wireName: r'builtin')
  static const ModelSourceSwitchRequestKindEnum builtin = _$modelSourceSwitchRequestKindEnum_builtin;
  @BuiltValueEnumConst(wireName: r'url')
  static const ModelSourceSwitchRequestKindEnum url = _$modelSourceSwitchRequestKindEnum_url;
  @BuiltValueEnumConst(wireName: r'file')
  static const ModelSourceSwitchRequestKindEnum file = _$modelSourceSwitchRequestKindEnum_file;

  static Serializer<ModelSourceSwitchRequestKindEnum> get serializer => _$modelSourceSwitchRequestKindEnumSerializer;

  const ModelSourceSwitchRequestKindEnum._(String name): super(name);

  static BuiltSet<ModelSourceSwitchRequestKindEnum> get values => _$modelSourceSwitchRequestKindEnumValues;
  static ModelSourceSwitchRequestKindEnum valueOf(String name) => _$modelSourceSwitchRequestKindEnumValueOf(name);
}

