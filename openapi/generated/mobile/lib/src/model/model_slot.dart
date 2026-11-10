//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_collection/built_collection.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'model_slot.g.dart';

/// The functional slot a model fills. `face_detect`, `face_recognize`, `clip_embed` and `ocr` are the machine learning slots of the pipeline; `clip_text` is the search's semantic arm -- the CLIP text tower that embeds queries into the vision tower's joint embedding space (it produces no per-asset rows, so its rerun endpoint answers 400 `MODEL_INVALID`); a speech slot arrives later. 
class ModelSlot extends EnumClass {

  @BuiltValueEnumConst(wireName: r'face_detect')
  static const ModelSlot faceDetect = _$faceDetect;
  @BuiltValueEnumConst(wireName: r'face_recognize')
  static const ModelSlot faceRecognize = _$faceRecognize;
  @BuiltValueEnumConst(wireName: r'clip_embed')
  static const ModelSlot clipEmbed = _$clipEmbed;
  @BuiltValueEnumConst(wireName: r'clip_text')
  static const ModelSlot clipText = _$clipText;
  @BuiltValueEnumConst(wireName: r'ocr')
  static const ModelSlot ocr = _$ocr;

  static Serializer<ModelSlot> get serializer => _$modelSlotSerializer;

  const ModelSlot._(String name): super(name);

  static BuiltSet<ModelSlot> get values => _$values;
  static ModelSlot valueOf(String name) => _$valueOf(name);
}

/// Optionally, enum_class can generate a mixin to go with your enum for use
/// with Angular. It exposes your enum constants as getters. So, if you mix it
/// in to your Dart component class, the values become available to the
/// corresponding Angular template.
///
/// Trigger mixin generation by writing a line like this one next to your enum.
abstract class ModelSlotMixin = Object with _$ModelSlotMixin;

