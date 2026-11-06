//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'place_facet_count.g.dart';

/// One place of the places facet.
///
/// Properties:
/// * [placeId] - The GeoNames id (the round-trip filter value).
/// * [name] - The locale display name, canonical fallback.
/// * [country] - The ISO country code.
/// * [assetCount] - The number of assets of the current filtered set (minus the place filter) resolved to this place. 
@BuiltValue()
abstract class PlaceFacetCount implements Built<PlaceFacetCount, PlaceFacetCountBuilder> {
  /// The GeoNames id (the round-trip filter value).
  @BuiltValueField(wireName: r'placeId')
  int get placeId;

  /// The locale display name, canonical fallback.
  @BuiltValueField(wireName: r'name')
  String get name;

  /// The ISO country code.
  @BuiltValueField(wireName: r'country')
  String get country;

  /// The number of assets of the current filtered set (minus the place filter) resolved to this place. 
  @BuiltValueField(wireName: r'assetCount')
  int get assetCount;

  PlaceFacetCount._();

  factory PlaceFacetCount([void updates(PlaceFacetCountBuilder b)]) = _$PlaceFacetCount;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(PlaceFacetCountBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<PlaceFacetCount> get serializer => _$PlaceFacetCountSerializer();
}

class _$PlaceFacetCountSerializer implements PrimitiveSerializer<PlaceFacetCount> {
  @override
  final Iterable<Type> types = const [PlaceFacetCount, _$PlaceFacetCount];

  @override
  final String wireName = r'PlaceFacetCount';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    PlaceFacetCount object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'placeId';
    yield serializers.serialize(
      object.placeId,
      specifiedType: const FullType(int),
    );
    yield r'name';
    yield serializers.serialize(
      object.name,
      specifiedType: const FullType(String),
    );
    yield r'country';
    yield serializers.serialize(
      object.country,
      specifiedType: const FullType(String),
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
    PlaceFacetCount object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required PlaceFacetCountBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'placeId':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(int),
          ) as int;
          result.placeId = valueDes;
          break;
        case r'name':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.name = valueDes;
          break;
        case r'country':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.country = valueDes;
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
  PlaceFacetCount deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = PlaceFacetCountBuilder();
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


