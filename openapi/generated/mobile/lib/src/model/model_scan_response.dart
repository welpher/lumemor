//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:lumemor_api/src/model/model_response.dart';
import 'package:built_collection/built_collection.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'model_scan_response.g.dart';

/// The custom model scan's report: every model directory the scan verified and registered, in scan order. Nothing is activated. 
///
/// Properties:
/// * [registered] - The models the scan registered, in scan order. 
@BuiltValue()
abstract class ModelScanResponse implements Built<ModelScanResponse, ModelScanResponseBuilder> {
  /// The models the scan registered, in scan order. 
  @BuiltValueField(wireName: r'registered')
  BuiltList<ModelResponse> get registered;

  ModelScanResponse._();

  factory ModelScanResponse([void updates(ModelScanResponseBuilder b)]) = _$ModelScanResponse;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(ModelScanResponseBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<ModelScanResponse> get serializer => _$ModelScanResponseSerializer();
}

class _$ModelScanResponseSerializer implements PrimitiveSerializer<ModelScanResponse> {
  @override
  final Iterable<Type> types = const [ModelScanResponse, _$ModelScanResponse];

  @override
  final String wireName = r'ModelScanResponse';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    ModelScanResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'registered';
    yield serializers.serialize(
      object.registered,
      specifiedType: const FullType(BuiltList, [FullType(ModelResponse)]),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    ModelScanResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required ModelScanResponseBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'registered':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(BuiltList, [FullType(ModelResponse)]),
          ) as BuiltList<ModelResponse>;
          result.registered.replace(valueDes);
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  ModelScanResponse deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = ModelScanResponseBuilder();
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


