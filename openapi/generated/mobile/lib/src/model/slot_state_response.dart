//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:lumemor_api/src/model/model_slot.dart';
import 'package:lumemor_api/src/model/slot_state_response_source.dart';
import 'package:lumemor_api/src/model/slot_state.dart';
import 'package:lumemor_api/src/model/slot_state_response_progress.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'slot_state_response.g.dart';

/// One pipeline slot's state row. `modelId` is the slot's active model's id, null when the slot is unfilled; `source` is null for an unfilled slot; `progress` is the in-flight download run's counts while `state` is `downloading` (and null otherwise -- the run is group-level, the card renders its percent). 
///
/// Properties:
/// * [slot] 
/// * [state] 
/// * [modelId] 
/// * [source_] 
/// * [progress] 
@BuiltValue()
abstract class SlotStateResponse implements Built<SlotStateResponse, SlotStateResponseBuilder> {
  @BuiltValueField(wireName: r'slot')
  ModelSlot get slot;
  // enum slotEnum {  face_detect,  face_recognize,  clip_embed,  clip_text,  ocr,  };

  @BuiltValueField(wireName: r'state')
  SlotState get state;
  // enum stateEnum {  active,  downloading,  not_downloaded,  failed,  };

  @BuiltValueField(wireName: r'modelId')
  String? get modelId;

  @BuiltValueField(wireName: r'source')
  SlotStateResponseSource? get source_;

  @BuiltValueField(wireName: r'progress')
  SlotStateResponseProgress? get progress;

  SlotStateResponse._();

  factory SlotStateResponse([void updates(SlotStateResponseBuilder b)]) = _$SlotStateResponse;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(SlotStateResponseBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<SlotStateResponse> get serializer => _$SlotStateResponseSerializer();
}

class _$SlotStateResponseSerializer implements PrimitiveSerializer<SlotStateResponse> {
  @override
  final Iterable<Type> types = const [SlotStateResponse, _$SlotStateResponse];

  @override
  final String wireName = r'SlotStateResponse';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    SlotStateResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'slot';
    yield serializers.serialize(
      object.slot,
      specifiedType: const FullType(ModelSlot),
    );
    yield r'state';
    yield serializers.serialize(
      object.state,
      specifiedType: const FullType(SlotState),
    );
    yield r'modelId';
    yield object.modelId == null ? null : serializers.serialize(
      object.modelId,
      specifiedType: const FullType.nullable(String),
    );
    yield r'source';
    yield object.source_ == null ? null : serializers.serialize(
      object.source_,
      specifiedType: const FullType.nullable(SlotStateResponseSource),
    );
    yield r'progress';
    yield object.progress == null ? null : serializers.serialize(
      object.progress,
      specifiedType: const FullType.nullable(SlotStateResponseProgress),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    SlotStateResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required SlotStateResponseBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'slot':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(ModelSlot),
          ) as ModelSlot;
          result.slot = valueDes;
          break;
        case r'state':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(SlotState),
          ) as SlotState;
          result.state = valueDes;
          break;
        case r'modelId':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(String),
          ) as String?;
          if (valueDes == null) continue;
          result.modelId = valueDes;
          break;
        case r'source':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(SlotStateResponseSource),
          ) as SlotStateResponseSource?;
          if (valueDes == null) continue;
          result.source_.replace(valueDes);
          break;
        case r'progress':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(SlotStateResponseProgress),
          ) as SlotStateResponseProgress?;
          if (valueDes == null) continue;
          result.progress.replace(valueDes);
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  SlotStateResponse deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = SlotStateResponseBuilder();
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


