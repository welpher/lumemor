//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'derivatives_summary_response.g.dart';

/// The image card's cost line: the library's image count and the declared estimate of a full rebuild at the current settings and dial. The estimate is the documented plan-of-record number (one second per asset, divided by the dial's current mapping), never a measurement. 
///
/// Properties:
/// * [imageCount] - The image assets the rebuild would visit. Video assets are not counted: the rebuild re-derives image serving copies only. 
/// * [estimateMinutes] - The declared estimate of a full rebuild, in whole minutes, floored at one. 
@BuiltValue()
abstract class DerivativesSummaryResponse implements Built<DerivativesSummaryResponse, DerivativesSummaryResponseBuilder> {
  /// The image assets the rebuild would visit. Video assets are not counted: the rebuild re-derives image serving copies only. 
  @BuiltValueField(wireName: r'imageCount')
  int get imageCount;

  /// The declared estimate of a full rebuild, in whole minutes, floored at one. 
  @BuiltValueField(wireName: r'estimateMinutes')
  int get estimateMinutes;

  DerivativesSummaryResponse._();

  factory DerivativesSummaryResponse([void updates(DerivativesSummaryResponseBuilder b)]) = _$DerivativesSummaryResponse;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(DerivativesSummaryResponseBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<DerivativesSummaryResponse> get serializer => _$DerivativesSummaryResponseSerializer();
}

class _$DerivativesSummaryResponseSerializer implements PrimitiveSerializer<DerivativesSummaryResponse> {
  @override
  final Iterable<Type> types = const [DerivativesSummaryResponse, _$DerivativesSummaryResponse];

  @override
  final String wireName = r'DerivativesSummaryResponse';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    DerivativesSummaryResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'imageCount';
    yield serializers.serialize(
      object.imageCount,
      specifiedType: const FullType(int),
    );
    yield r'estimateMinutes';
    yield serializers.serialize(
      object.estimateMinutes,
      specifiedType: const FullType(int),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    DerivativesSummaryResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required DerivativesSummaryResponseBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'imageCount':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(int),
          ) as int;
          result.imageCount = valueDes;
          break;
        case r'estimateMinutes':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(int),
          ) as int;
          result.estimateMinutes = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  DerivativesSummaryResponse deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = DerivativesSummaryResponseBuilder();
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


