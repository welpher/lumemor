//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'model_probe_accepted_response.g.dart';

/// The 202 answer of a probe POST: the identifier to poll. 
///
/// Properties:
/// * [probeId] 
@BuiltValue()
abstract class ModelProbeAcceptedResponse implements Built<ModelProbeAcceptedResponse, ModelProbeAcceptedResponseBuilder> {
  @BuiltValueField(wireName: r'probeId')
  String get probeId;

  ModelProbeAcceptedResponse._();

  factory ModelProbeAcceptedResponse([void updates(ModelProbeAcceptedResponseBuilder b)]) = _$ModelProbeAcceptedResponse;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(ModelProbeAcceptedResponseBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<ModelProbeAcceptedResponse> get serializer => _$ModelProbeAcceptedResponseSerializer();
}

class _$ModelProbeAcceptedResponseSerializer implements PrimitiveSerializer<ModelProbeAcceptedResponse> {
  @override
  final Iterable<Type> types = const [ModelProbeAcceptedResponse, _$ModelProbeAcceptedResponse];

  @override
  final String wireName = r'ModelProbeAcceptedResponse';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    ModelProbeAcceptedResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'probeId';
    yield serializers.serialize(
      object.probeId,
      specifiedType: const FullType(String),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    ModelProbeAcceptedResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required ModelProbeAcceptedResponseBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'probeId':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.probeId = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  ModelProbeAcceptedResponse deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = ModelProbeAcceptedResponseBuilder();
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


