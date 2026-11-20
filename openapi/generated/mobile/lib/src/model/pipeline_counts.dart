//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:lumemor_api/src/model/pipeline_job_counts.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'pipeline_counts.g.dart';

/// The live queue counts of each pipeline kind, read from the queue's job table. The keys name the pipeline workers: the three media jobs and the four machine learning jobs. 
///
/// Properties:
/// * [thumbnail] 
/// * [metadata] 
/// * [transcode] 
/// * [faceDetect] 
/// * [faceRecognize] 
/// * [clipEmbed] 
/// * [ocr] 
@BuiltValue()
abstract class PipelineCounts implements Built<PipelineCounts, PipelineCountsBuilder> {
  @BuiltValueField(wireName: r'thumbnail')
  PipelineJobCounts get thumbnail;

  @BuiltValueField(wireName: r'metadata')
  PipelineJobCounts get metadata;

  @BuiltValueField(wireName: r'transcode')
  PipelineJobCounts get transcode;

  @BuiltValueField(wireName: r'faceDetect')
  PipelineJobCounts get faceDetect;

  @BuiltValueField(wireName: r'faceRecognize')
  PipelineJobCounts get faceRecognize;

  @BuiltValueField(wireName: r'clipEmbed')
  PipelineJobCounts get clipEmbed;

  @BuiltValueField(wireName: r'ocr')
  PipelineJobCounts get ocr;

  PipelineCounts._();

  factory PipelineCounts([void updates(PipelineCountsBuilder b)]) = _$PipelineCounts;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(PipelineCountsBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<PipelineCounts> get serializer => _$PipelineCountsSerializer();
}

class _$PipelineCountsSerializer implements PrimitiveSerializer<PipelineCounts> {
  @override
  final Iterable<Type> types = const [PipelineCounts, _$PipelineCounts];

  @override
  final String wireName = r'PipelineCounts';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    PipelineCounts object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'thumbnail';
    yield serializers.serialize(
      object.thumbnail,
      specifiedType: const FullType(PipelineJobCounts),
    );
    yield r'metadata';
    yield serializers.serialize(
      object.metadata,
      specifiedType: const FullType(PipelineJobCounts),
    );
    yield r'transcode';
    yield serializers.serialize(
      object.transcode,
      specifiedType: const FullType(PipelineJobCounts),
    );
    yield r'faceDetect';
    yield serializers.serialize(
      object.faceDetect,
      specifiedType: const FullType(PipelineJobCounts),
    );
    yield r'faceRecognize';
    yield serializers.serialize(
      object.faceRecognize,
      specifiedType: const FullType(PipelineJobCounts),
    );
    yield r'clipEmbed';
    yield serializers.serialize(
      object.clipEmbed,
      specifiedType: const FullType(PipelineJobCounts),
    );
    yield r'ocr';
    yield serializers.serialize(
      object.ocr,
      specifiedType: const FullType(PipelineJobCounts),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    PipelineCounts object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required PipelineCountsBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'thumbnail':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(PipelineJobCounts),
          ) as PipelineJobCounts;
          result.thumbnail.replace(valueDes);
          break;
        case r'metadata':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(PipelineJobCounts),
          ) as PipelineJobCounts;
          result.metadata.replace(valueDes);
          break;
        case r'transcode':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(PipelineJobCounts),
          ) as PipelineJobCounts;
          result.transcode.replace(valueDes);
          break;
        case r'faceDetect':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(PipelineJobCounts),
          ) as PipelineJobCounts;
          result.faceDetect.replace(valueDes);
          break;
        case r'faceRecognize':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(PipelineJobCounts),
          ) as PipelineJobCounts;
          result.faceRecognize.replace(valueDes);
          break;
        case r'clipEmbed':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(PipelineJobCounts),
          ) as PipelineJobCounts;
          result.clipEmbed.replace(valueDes);
          break;
        case r'ocr':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(PipelineJobCounts),
          ) as PipelineJobCounts;
          result.ocr.replace(valueDes);
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  PipelineCounts deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = PipelineCountsBuilder();
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


