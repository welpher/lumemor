//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_collection/built_collection.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'job_run_kind.g.dart';

/// The kinds of job runs the instance records. The model download is one run; each machine learning slot's rerun is its own kind, named after the slot it redoes; the place re-resolution is the places-data-update path's kind; the album rescan is the directory album's source-reconciliation run; the derivative rebuild is the image card's explicit re-derivation of the serving copies at the current settings. 
class JobRunKind extends EnumClass {

  @BuiltValueEnumConst(wireName: r'model_download')
  static const JobRunKind modelDownload = _$modelDownload;
  @BuiltValueEnumConst(wireName: r'face_detect_rerun')
  static const JobRunKind faceDetectRerun = _$faceDetectRerun;
  @BuiltValueEnumConst(wireName: r'face_recognize_rerun')
  static const JobRunKind faceRecognizeRerun = _$faceRecognizeRerun;
  @BuiltValueEnumConst(wireName: r'clip_embed_rerun')
  static const JobRunKind clipEmbedRerun = _$clipEmbedRerun;
  @BuiltValueEnumConst(wireName: r'ocr_rerun')
  static const JobRunKind ocrRerun = _$ocrRerun;
  @BuiltValueEnumConst(wireName: r'places_rerun')
  static const JobRunKind placesRerun = _$placesRerun;
  @BuiltValueEnumConst(wireName: r'album_rescan')
  static const JobRunKind albumRescan = _$albumRescan;
  @BuiltValueEnumConst(wireName: r'rebuild_derivatives')
  static const JobRunKind rebuildDerivatives = _$rebuildDerivatives;

  static Serializer<JobRunKind> get serializer => _$jobRunKindSerializer;

  const JobRunKind._(String name): super(name);

  static BuiltSet<JobRunKind> get values => _$values;
  static JobRunKind valueOf(String name) => _$valueOf(name);
}

/// Optionally, enum_class can generate a mixin to go with your enum for use
/// with Angular. It exposes your enum constants as getters. So, if you mix it
/// in to your Dart component class, the values become available to the
/// corresponding Angular template.
///
/// Trigger mixin generation by writing a line like this one next to your enum.
abstract class JobRunKindMixin = Object with _$JobRunKindMixin;

