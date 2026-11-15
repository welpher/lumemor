//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'asset_coordinates.g.dart';

/// One geographic position in degrees: the pair the info panel's mini map and coordinate line render. Both halves always present -- half a coordinate is no location. 
///
/// Properties:
/// * [lat] - Latitude in degrees (-90..90).
/// * [lon] - Longitude in degrees (-180..180).
@BuiltValue()
abstract class AssetCoordinates implements Built<AssetCoordinates, AssetCoordinatesBuilder> {
  /// Latitude in degrees (-90..90).
  @BuiltValueField(wireName: r'lat')
  double get lat;

  /// Longitude in degrees (-180..180).
  @BuiltValueField(wireName: r'lon')
  double get lon;

  AssetCoordinates._();

  factory AssetCoordinates([void updates(AssetCoordinatesBuilder b)]) = _$AssetCoordinates;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(AssetCoordinatesBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<AssetCoordinates> get serializer => _$AssetCoordinatesSerializer();
}

class _$AssetCoordinatesSerializer implements PrimitiveSerializer<AssetCoordinates> {
  @override
  final Iterable<Type> types = const [AssetCoordinates, _$AssetCoordinates];

  @override
  final String wireName = r'AssetCoordinates';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    AssetCoordinates object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'lat';
    yield serializers.serialize(
      object.lat,
      specifiedType: const FullType(double),
    );
    yield r'lon';
    yield serializers.serialize(
      object.lon,
      specifiedType: const FullType(double),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    AssetCoordinates object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required AssetCoordinatesBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'lat':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(double),
          ) as double;
          result.lat = valueDes;
          break;
        case r'lon':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(double),
          ) as double;
          result.lon = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  AssetCoordinates deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = AssetCoordinatesBuilder();
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


