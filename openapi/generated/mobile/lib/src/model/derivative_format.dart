//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_collection/built_collection.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'derivative_format.g.dart';

/// The serving copies' encoding. `avif` stores smaller and encodes slower (on a weak machine it makes first browsing slower); `webp` encodes faster and stores a little larger. 
class DerivativeFormat extends EnumClass {

  @BuiltValueEnumConst(wireName: r'avif')
  static const DerivativeFormat avif = _$avif;
  @BuiltValueEnumConst(wireName: r'webp')
  static const DerivativeFormat webp = _$webp;

  static Serializer<DerivativeFormat> get serializer => _$derivativeFormatSerializer;

  const DerivativeFormat._(String name): super(name);

  static BuiltSet<DerivativeFormat> get values => _$values;
  static DerivativeFormat valueOf(String name) => _$valueOf(name);
}

/// Optionally, enum_class can generate a mixin to go with your enum for use
/// with Angular. It exposes your enum constants as getters. So, if you mix it
/// in to your Dart component class, the values become available to the
/// corresponding Angular template.
///
/// Trigger mixin generation by writing a line like this one next to your enum.
abstract class DerivativeFormatMixin = Object with _$DerivativeFormatMixin;

