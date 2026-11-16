//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'model_params.g.dart';

/// The pipeline's steering params (E41): the values the ML jobs read at their own dispatch. The set is honest by contract -- v1 carries exactly the two knobs the pipeline actually reads, and it grows only as the pipeline grows (a slot without a knob renders no empty block).  A change takes effect on the NEXT dispatch: each job reads the params once at its own start, so work already in flight runs to completion on the values it read; no in-flight job is re-steered. A corrupt stored value (a row the save endpoint's validation should have kept honest) answers the pipeline default on the read -- 0.6 for the minimum score, 0.55 for the clustering threshold -- with the corruption logged server-side. 
///
/// Properties:
/// * [faceDetectMinScore] - The face detection's minimum confidence, 0 to 1 inclusive: a candidate scoring below it never enters the library. The shipped default is the pipeline's own 0.6; the admin value overrides the active model's manifest entry for this knob. Contract granularity is steps of 0.05 (the page's stepper); the domain is what validates. 
/// * [faceRecognizeThreshold] - The face recognition's clustering threshold, 0 to 1 inclusive: two embeddings whose cosine distance is below it belong to one person. The shipped default is the pipeline's own 0.55; the admin value overrides the active model's manifest entry for this knob. Contract granularity is steps of 0.05 (the page's stepper); the domain is what validates. 
@BuiltValue()
abstract class ModelParams implements Built<ModelParams, ModelParamsBuilder> {
  /// The face detection's minimum confidence, 0 to 1 inclusive: a candidate scoring below it never enters the library. The shipped default is the pipeline's own 0.6; the admin value overrides the active model's manifest entry for this knob. Contract granularity is steps of 0.05 (the page's stepper); the domain is what validates. 
  @BuiltValueField(wireName: r'faceDetectMinScore')
  num get faceDetectMinScore;

  /// The face recognition's clustering threshold, 0 to 1 inclusive: two embeddings whose cosine distance is below it belong to one person. The shipped default is the pipeline's own 0.55; the admin value overrides the active model's manifest entry for this knob. Contract granularity is steps of 0.05 (the page's stepper); the domain is what validates. 
  @BuiltValueField(wireName: r'faceRecognizeThreshold')
  num get faceRecognizeThreshold;

  ModelParams._();

  factory ModelParams([void updates(ModelParamsBuilder b)]) = _$ModelParams;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(ModelParamsBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<ModelParams> get serializer => _$ModelParamsSerializer();
}

class _$ModelParamsSerializer implements PrimitiveSerializer<ModelParams> {
  @override
  final Iterable<Type> types = const [ModelParams, _$ModelParams];

  @override
  final String wireName = r'ModelParams';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    ModelParams object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'faceDetectMinScore';
    yield serializers.serialize(
      object.faceDetectMinScore,
      specifiedType: const FullType(num),
    );
    yield r'faceRecognizeThreshold';
    yield serializers.serialize(
      object.faceRecognizeThreshold,
      specifiedType: const FullType(num),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    ModelParams object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required ModelParamsBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'faceDetectMinScore':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(num),
          ) as num;
          result.faceDetectMinScore = valueDes;
          break;
        case r'faceRecognizeThreshold':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(num),
          ) as num;
          result.faceRecognizeThreshold = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  ModelParams deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = ModelParamsBuilder();
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


