//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_collection/built_collection.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'transcode_preset.g.dart';

/// The transcode job's re-encode preset for new video ingests: `auto` keeps the source resolution, the numeric presets scale down to the named vertical bound, and `off` skips the re-encode entirely (the original stays in place and downloadable; the video's pipeline still completes). 
class TranscodePreset extends EnumClass {

  @BuiltValueEnumConst(wireName: r'auto')
  static const TranscodePreset auto = _$auto;
  @BuiltValueEnumConst(wireName: r'1080p')
  static const TranscodePreset n1080p = _$n1080p;
  @BuiltValueEnumConst(wireName: r'720p')
  static const TranscodePreset n720p = _$n720p;
  @BuiltValueEnumConst(wireName: r'480p')
  static const TranscodePreset n480p = _$n480p;
  @BuiltValueEnumConst(wireName: r'off')
  static const TranscodePreset off = _$off;

  static Serializer<TranscodePreset> get serializer => _$transcodePresetSerializer;

  const TranscodePreset._(String name): super(name);

  static BuiltSet<TranscodePreset> get values => _$values;
  static TranscodePreset valueOf(String name) => _$valueOf(name);
}

/// Optionally, enum_class can generate a mixin to go with your enum for use
/// with Angular. It exposes your enum constants as getters. So, if you mix it
/// in to your Dart component class, the values become available to the
/// corresponding Angular template.
///
/// Trigger mixin generation by writing a line like this one next to your enum.
abstract class TranscodePresetMixin = Object with _$TranscodePresetMixin;

