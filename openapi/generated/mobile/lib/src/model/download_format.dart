//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_collection/built_collection.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'download_format.g.dart';

/// The account's preferred archive format of batch and whole-album downloads. 
class DownloadFormat extends EnumClass {

  @BuiltValueEnumConst(wireName: r'zip')
  static const DownloadFormat zip = _$zip;
  @BuiltValueEnumConst(wireName: r'tar')
  static const DownloadFormat tar = _$tar;

  static Serializer<DownloadFormat> get serializer => _$downloadFormatSerializer;

  const DownloadFormat._(String name): super(name);

  static BuiltSet<DownloadFormat> get values => _$values;
  static DownloadFormat valueOf(String name) => _$valueOf(name);
}

/// Optionally, enum_class can generate a mixin to go with your enum for use
/// with Angular. It exposes your enum constants as getters. So, if you mix it
/// in to your Dart component class, the values become available to the
/// corresponding Angular template.
///
/// Trigger mixin generation by writing a line like this one next to your enum.
abstract class DownloadFormatMixin = Object with _$DownloadFormatMixin;

