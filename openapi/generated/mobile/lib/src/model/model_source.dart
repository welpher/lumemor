//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_collection/built_collection.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'model_source.g.dart';

/// Where a model's registration came from: `downloaded` models landed through the download endpoint's manifest-pinned group; `custom` models were registered by the scan of a directory an administrator placed. 
class ModelSource extends EnumClass {

  @BuiltValueEnumConst(wireName: r'downloaded')
  static const ModelSource downloaded = _$downloaded;
  @BuiltValueEnumConst(wireName: r'custom')
  static const ModelSource custom = _$custom;

  static Serializer<ModelSource> get serializer => _$modelSourceSerializer;

  const ModelSource._(String name): super(name);

  static BuiltSet<ModelSource> get values => _$values;
  static ModelSource valueOf(String name) => _$valueOf(name);
}

/// Optionally, enum_class can generate a mixin to go with your enum for use
/// with Angular. It exposes your enum constants as getters. So, if you mix it
/// in to your Dart component class, the values become available to the
/// corresponding Angular template.
///
/// Trigger mixin generation by writing a line like this one next to your enum.
abstract class ModelSourceMixin = Object with _$ModelSourceMixin;

