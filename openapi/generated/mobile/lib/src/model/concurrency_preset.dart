//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_collection/built_collection.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'concurrency_preset.g.dart';

/// The concurrency dial's three steps, mapped to worker counts: `light` runs one job at a time, `standard` runs half the cores, `full` runs all but one. The panel never asks for a number -- the mapping is the explanation line. 
class ConcurrencyPreset extends EnumClass {

  @BuiltValueEnumConst(wireName: r'light')
  static const ConcurrencyPreset light = _$light;
  @BuiltValueEnumConst(wireName: r'standard')
  static const ConcurrencyPreset standard = _$standard;
  @BuiltValueEnumConst(wireName: r'full')
  static const ConcurrencyPreset full = _$full;

  static Serializer<ConcurrencyPreset> get serializer => _$concurrencyPresetSerializer;

  const ConcurrencyPreset._(String name): super(name);

  static BuiltSet<ConcurrencyPreset> get values => _$values;
  static ConcurrencyPreset valueOf(String name) => _$valueOf(name);
}

/// Optionally, enum_class can generate a mixin to go with your enum for use
/// with Angular. It exposes your enum constants as getters. So, if you mix it
/// in to your Dart component class, the values become available to the
/// corresponding Angular template.
///
/// Trigger mixin generation by writing a line like this one next to your enum.
abstract class ConcurrencyPresetMixin = Object with _$ConcurrencyPresetMixin;

