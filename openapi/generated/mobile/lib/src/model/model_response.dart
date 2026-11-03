//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:lumemor_api/src/model/model_slot.dart';
import 'package:built_collection/built_collection.dart';
import 'package:lumemor_api/src/model/model_response_input_size.dart';
import 'package:lumemor_api/src/model/model_source.dart';
import 'package:built_value/json_object.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'model_response.g.dart';

/// One registered model as the admin surface returns it: the database row's identity, source, dimension and params, the on-disk manifest's license and input geometry, the activation flag, and the presence probe of the model's files. The pinned sha256 digests and the file paths are server-side business and never leave the server. 
///
/// Properties:
/// * [id] - The model's stable slug identifier -- also the name of its directory under the models dir. 
/// * [slot] 
/// * [version] - The upstream release tag of the model family, as the manifest declares it. 
/// * [source_] 
/// * [license] - The upstream license identifier, as the model's manifest declares it (empty when the manifest is unreadable -- the row still lists so the administrator can clean it up). 
/// * [dim] - The embedding dimension the model emits. Null for models that do not embed (detectors, OCR). 
/// * [inputSize] 
/// * [isActive] - True when the model is the active one of its slot. At most one model per slot is active. 
/// * [filesPresent] - True when the model's on-disk manifest is readable and every file it pins exists -- a presence probe only. The sha256 verification runs at activation and load time, not here. 
/// * [params] - The model's inference defaults (thresholds, normalization constants) as its manifest declares them; empty when it declares none. 
@BuiltValue()
abstract class ModelResponse implements Built<ModelResponse, ModelResponseBuilder> {
  /// The model's stable slug identifier -- also the name of its directory under the models dir. 
  @BuiltValueField(wireName: r'id')
  String get id;

  @BuiltValueField(wireName: r'slot')
  ModelSlot get slot;
  // enum slotEnum {  face_detect,  face_recognize,  clip_embed,  ocr,  };

  /// The upstream release tag of the model family, as the manifest declares it. 
  @BuiltValueField(wireName: r'version')
  String get version;

  @BuiltValueField(wireName: r'source')
  ModelSource get source_;
  // enum source_Enum {  downloaded,  custom,  };

  /// The upstream license identifier, as the model's manifest declares it (empty when the manifest is unreadable -- the row still lists so the administrator can clean it up). 
  @BuiltValueField(wireName: r'license')
  String get license;

  /// The embedding dimension the model emits. Null for models that do not embed (detectors, OCR). 
  @BuiltValueField(wireName: r'dim')
  int? get dim;

  @BuiltValueField(wireName: r'inputSize')
  ModelResponseInputSize? get inputSize;

  /// True when the model is the active one of its slot. At most one model per slot is active. 
  @BuiltValueField(wireName: r'isActive')
  bool get isActive;

  /// True when the model's on-disk manifest is readable and every file it pins exists -- a presence probe only. The sha256 verification runs at activation and load time, not here. 
  @BuiltValueField(wireName: r'filesPresent')
  bool get filesPresent;

  /// The model's inference defaults (thresholds, normalization constants) as its manifest declares them; empty when it declares none. 
  @BuiltValueField(wireName: r'params')
  BuiltMap<String, JsonObject?> get params;

  ModelResponse._();

  factory ModelResponse([void updates(ModelResponseBuilder b)]) = _$ModelResponse;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(ModelResponseBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<ModelResponse> get serializer => _$ModelResponseSerializer();
}

class _$ModelResponseSerializer implements PrimitiveSerializer<ModelResponse> {
  @override
  final Iterable<Type> types = const [ModelResponse, _$ModelResponse];

  @override
  final String wireName = r'ModelResponse';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    ModelResponse object, {
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
    yield r'version';
    yield serializers.serialize(
      object.version,
      specifiedType: const FullType(String),
    );
    yield r'source';
    yield serializers.serialize(
      object.source_,
      specifiedType: const FullType(ModelSource),
    );
    yield r'license';
    yield serializers.serialize(
      object.license,
      specifiedType: const FullType(String),
    );
    yield r'dim';
    yield object.dim == null ? null : serializers.serialize(
      object.dim,
      specifiedType: const FullType.nullable(int),
    );
    yield r'inputSize';
    yield object.inputSize == null ? null : serializers.serialize(
      object.inputSize,
      specifiedType: const FullType.nullable(ModelResponseInputSize),
    );
    yield r'isActive';
    yield serializers.serialize(
      object.isActive,
      specifiedType: const FullType(bool),
    );
    yield r'filesPresent';
    yield serializers.serialize(
      object.filesPresent,
      specifiedType: const FullType(bool),
    );
    yield r'params';
    yield serializers.serialize(
      object.params,
      specifiedType: const FullType(BuiltMap, [FullType(String), FullType.nullable(JsonObject)]),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    ModelResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required ModelResponseBuilder result,
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
        case r'version':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.version = valueDes;
          break;
        case r'source':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(ModelSource),
          ) as ModelSource;
          result.source_ = valueDes;
          break;
        case r'license':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.license = valueDes;
          break;
        case r'dim':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(int),
          ) as int?;
          if (valueDes == null) continue;
          result.dim = valueDes;
          break;
        case r'inputSize':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(ModelResponseInputSize),
          ) as ModelResponseInputSize?;
          if (valueDes == null) continue;
          result.inputSize.replace(valueDes);
          break;
        case r'isActive':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(bool),
          ) as bool;
          result.isActive = valueDes;
          break;
        case r'filesPresent':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(bool),
          ) as bool;
          result.filesPresent = valueDes;
          break;
        case r'params':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(BuiltMap, [FullType(String), FullType.nullable(JsonObject)]),
          ) as BuiltMap<String, JsonObject?>;
          result.params.replace(valueDes);
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  ModelResponse deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = ModelResponseBuilder();
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


