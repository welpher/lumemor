//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:lumemor_api/src/model/model_response.dart';
import 'package:built_collection/built_collection.dart';
import 'package:lumemor_api/src/model/model_catalog_entry.dart';
import 'package:lumemor_api/src/model/slot_state_response.dart';
import 'package:lumemor_api/src/model/job_run_response.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'model_list_response.g.dart';

/// The model management listing: every registered model plus the download run summary, the per-slot state view the page's cards render, and the manifest catalog the switch dialog's official group renders. 
///
/// Properties:
/// * [models] - Every registered model, grouped by slot then id. 
/// * [download] - The most recent `model_download` run of whatever state, or null before the first download was ever started. 
/// * [slots] - One row per pipeline slot (the contract's enum order): the four-state view the page's cards render, the slot's active model, its source facts and the download run's progress while a run is in flight. 
/// * [official] - The manifest's catalog: the validated list the switch dialog's official group renders, every entry with its download state. The image is not the entry -- the mirror carries no weights; `downloaded` is the presence probe and the digest check stays the activation's business. 
@BuiltValue()
abstract class ModelListResponse implements Built<ModelListResponse, ModelListResponseBuilder> {
  /// Every registered model, grouped by slot then id. 
  @BuiltValueField(wireName: r'models')
  BuiltList<ModelResponse> get models;

  /// The most recent `model_download` run of whatever state, or null before the first download was ever started. 
  @BuiltValueField(wireName: r'download')
  JobRunResponse get download;

  /// One row per pipeline slot (the contract's enum order): the four-state view the page's cards render, the slot's active model, its source facts and the download run's progress while a run is in flight. 
  @BuiltValueField(wireName: r'slots')
  BuiltList<SlotStateResponse> get slots;

  /// The manifest's catalog: the validated list the switch dialog's official group renders, every entry with its download state. The image is not the entry -- the mirror carries no weights; `downloaded` is the presence probe and the digest check stays the activation's business. 
  @BuiltValueField(wireName: r'official')
  BuiltList<ModelCatalogEntry> get official;

  ModelListResponse._();

  factory ModelListResponse([void updates(ModelListResponseBuilder b)]) = _$ModelListResponse;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(ModelListResponseBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<ModelListResponse> get serializer => _$ModelListResponseSerializer();
}

class _$ModelListResponseSerializer implements PrimitiveSerializer<ModelListResponse> {
  @override
  final Iterable<Type> types = const [ModelListResponse, _$ModelListResponse];

  @override
  final String wireName = r'ModelListResponse';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    ModelListResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'models';
    yield serializers.serialize(
      object.models,
      specifiedType: const FullType(BuiltList, [FullType(ModelResponse)]),
    );
    yield r'download';
    yield serializers.serialize(
      object.download,
      specifiedType: const FullType(JobRunResponse),
    );
    yield r'slots';
    yield serializers.serialize(
      object.slots,
      specifiedType: const FullType(BuiltList, [FullType(SlotStateResponse)]),
    );
    yield r'official';
    yield serializers.serialize(
      object.official,
      specifiedType: const FullType(BuiltList, [FullType(ModelCatalogEntry)]),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    ModelListResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required ModelListResponseBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'models':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(BuiltList, [FullType(ModelResponse)]),
          ) as BuiltList<ModelResponse>;
          result.models.replace(valueDes);
          break;
        case r'download':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(JobRunResponse),
          ) as JobRunResponse;
          result.download.replace(valueDes);
          break;
        case r'slots':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(BuiltList, [FullType(SlotStateResponse)]),
          ) as BuiltList<SlotStateResponse>;
          result.slots.replace(valueDes);
          break;
        case r'official':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(BuiltList, [FullType(ModelCatalogEntry)]),
          ) as BuiltList<ModelCatalogEntry>;
          result.official.replace(valueDes);
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  ModelListResponse deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = ModelListResponseBuilder();
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


