//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'asset_place_ref.g.dart';

/// The asset detail's `place` face: the resolved city as the info panel renders it -- the deep link's key and the display name, nothing more (the place listing's counts and covers are the `GET /places` face's own). 
///
/// Properties:
/// * [placeId] - The GeoNames geonameid of the place -- the `places` row's primary key and the `assets.place_id` foreign key, the same identifier the place-filtered browse filters on. 
/// * [name] - The display name at the request's locale: the `place_names` row of that locale when the place data carries one, the GeoNames canonical (English or romanized) name otherwise. 
@BuiltValue()
abstract class AssetPlaceRef implements Built<AssetPlaceRef, AssetPlaceRefBuilder> {
  /// The GeoNames geonameid of the place -- the `places` row's primary key and the `assets.place_id` foreign key, the same identifier the place-filtered browse filters on. 
  @BuiltValueField(wireName: r'placeId')
  int get placeId;

  /// The display name at the request's locale: the `place_names` row of that locale when the place data carries one, the GeoNames canonical (English or romanized) name otherwise. 
  @BuiltValueField(wireName: r'name')
  String get name;

  AssetPlaceRef._();

  factory AssetPlaceRef([void updates(AssetPlaceRefBuilder b)]) = _$AssetPlaceRef;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(AssetPlaceRefBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<AssetPlaceRef> get serializer => _$AssetPlaceRefSerializer();
}

class _$AssetPlaceRefSerializer implements PrimitiveSerializer<AssetPlaceRef> {
  @override
  final Iterable<Type> types = const [AssetPlaceRef, _$AssetPlaceRef];

  @override
  final String wireName = r'AssetPlaceRef';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    AssetPlaceRef object, {
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
  }

  @override
  Object serialize(
    Serializers serializers,
    AssetPlaceRef object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required AssetPlaceRefBuilder result,
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
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  AssetPlaceRef deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = AssetPlaceRefBuilder();
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


