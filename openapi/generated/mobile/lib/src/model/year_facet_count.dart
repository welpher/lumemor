//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'year_facet_count.g.dart';

/// One year of the time-range facet.
///
/// Properties:
/// * [year] - The taken-at year.
/// * [assetCount] - The number of assets of the current filtered set (minus the time-range filter) taken in this year. 
@BuiltValue()
abstract class YearFacetCount implements Built<YearFacetCount, YearFacetCountBuilder> {
  /// The taken-at year.
  @BuiltValueField(wireName: r'year')
  int get year;

  /// The number of assets of the current filtered set (minus the time-range filter) taken in this year. 
  @BuiltValueField(wireName: r'assetCount')
  int get assetCount;

  YearFacetCount._();

  factory YearFacetCount([void updates(YearFacetCountBuilder b)]) = _$YearFacetCount;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(YearFacetCountBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<YearFacetCount> get serializer => _$YearFacetCountSerializer();
}

class _$YearFacetCountSerializer implements PrimitiveSerializer<YearFacetCount> {
  @override
  final Iterable<Type> types = const [YearFacetCount, _$YearFacetCount];

  @override
  final String wireName = r'YearFacetCount';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    YearFacetCount object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'year';
    yield serializers.serialize(
      object.year,
      specifiedType: const FullType(int),
    );
    yield r'assetCount';
    yield serializers.serialize(
      object.assetCount,
      specifiedType: const FullType(int),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    YearFacetCount object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required YearFacetCountBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'year':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(int),
          ) as int;
          result.year = valueDes;
          break;
        case r'assetCount':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(int),
          ) as int;
          result.assetCount = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  YearFacetCount deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = YearFacetCountBuilder();
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


