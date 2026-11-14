//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_collection/built_collection.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'slot_state.g.dart';

/// The four states a slot's card speaks. `active` is a slot running its model (the state badge renders as the model's own facts); `downloading` is a model download run in flight whose bytes this slot still waits for (the card shows the run's percent); `not_downloaded` is an unfilled slot -- or a filled one whose weights are gone with no run coming; `failed` is the same missing-weights state while the most recent download run failed (the card shows the retry). 
class SlotState extends EnumClass {

  @BuiltValueEnumConst(wireName: r'active')
  static const SlotState active = _$active;
  @BuiltValueEnumConst(wireName: r'downloading')
  static const SlotState downloading = _$downloading;
  @BuiltValueEnumConst(wireName: r'not_downloaded')
  static const SlotState notDownloaded = _$notDownloaded;
  @BuiltValueEnumConst(wireName: r'failed')
  static const SlotState failed = _$failed;

  static Serializer<SlotState> get serializer => _$slotStateSerializer;

  const SlotState._(String name): super(name);

  static BuiltSet<SlotState> get values => _$values;
  static SlotState valueOf(String name) => _$valueOf(name);
}

/// Optionally, enum_class can generate a mixin to go with your enum for use
/// with Angular. It exposes your enum constants as getters. So, if you mix it
/// in to your Dart component class, the values become available to the
/// corresponding Angular template.
///
/// Trigger mixin generation by writing a line like this one next to your enum.
abstract class SlotStateMixin = Object with _$SlotStateMixin;

