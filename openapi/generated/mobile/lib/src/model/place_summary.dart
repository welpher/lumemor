//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_collection/built_collection.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'place_summary.g.dart';

/// One city group of the caller's places list: a GeoNames place holding at least one caller-visible live asset. The count and the covers are the caller's honest view -- the same place can show different counts to different accounts. 
///
/// Properties:
/// * [placeId] - The GeoNames geonameid of the place -- the `places` row's primary key and the `assets.place_id` foreign key the ingestion pipeline's resolution writes. 
/// * [name] - The display name in the request's locale: the `place_names` row of that locale when the place data carries one, the GeoNames canonical (English or romanized) name otherwise. A locale switch changes this join only, never an asset row. 
/// * [country] - The place's ISO 3166-1 alpha-2 country code. The display country name is the client's business (its own locale table maps the code). 
/// * [assetCount] - The place's caller-visible live assets -- the honest per-caller count, not the global member count. 
/// * [latitude] - The place's latitude in degrees (the GeoNames record's own coordinate, never a photo's). It is the point the places map pins and clusters -- the map layer is a view over the same grouping, so no asset coordinate ever needs to leave the instance for it. 
/// * [longitude] - The place's longitude in degrees (the GeoNames record's own coordinate, never a photo's). 
/// * [takenAtFrom] - The earliest capture time among the place's caller-visible live members (RFC 3339 UTC). The city card's year span starts here; the listing's own ordering reads the other end. 
/// * [takenAtTo] - The latest capture time among the place's caller-visible live members (RFC 3339 UTC). It is the ordering key of the list -- the most recently photographed city leads -- and the city card's year span ends here. Equal to `takenAtFrom` when the place holds one member. 
/// * [representativeCoverAssetIds] - The place's cover candidates: its (up to) four most recent caller-visible live members, taken_at descending with the asset id as the tiebreak -- the wall's derived-cover discipline applied to city groups, newest first. 
@BuiltValue()
abstract class PlaceSummary implements Built<PlaceSummary, PlaceSummaryBuilder> {
  /// The GeoNames geonameid of the place -- the `places` row's primary key and the `assets.place_id` foreign key the ingestion pipeline's resolution writes. 
  @BuiltValueField(wireName: r'placeId')
  int get placeId;

  /// The display name in the request's locale: the `place_names` row of that locale when the place data carries one, the GeoNames canonical (English or romanized) name otherwise. A locale switch changes this join only, never an asset row. 
  @BuiltValueField(wireName: r'name')
  String get name;

  /// The place's ISO 3166-1 alpha-2 country code. The display country name is the client's business (its own locale table maps the code). 
  @BuiltValueField(wireName: r'country')
  String get country;

  /// The place's caller-visible live assets -- the honest per-caller count, not the global member count. 
  @BuiltValueField(wireName: r'assetCount')
  int get assetCount;

  /// The place's latitude in degrees (the GeoNames record's own coordinate, never a photo's). It is the point the places map pins and clusters -- the map layer is a view over the same grouping, so no asset coordinate ever needs to leave the instance for it. 
  @BuiltValueField(wireName: r'latitude')
  double get latitude;

  /// The place's longitude in degrees (the GeoNames record's own coordinate, never a photo's). 
  @BuiltValueField(wireName: r'longitude')
  double get longitude;

  /// The earliest capture time among the place's caller-visible live members (RFC 3339 UTC). The city card's year span starts here; the listing's own ordering reads the other end. 
  @BuiltValueField(wireName: r'takenAtFrom')
  DateTime get takenAtFrom;

  /// The latest capture time among the place's caller-visible live members (RFC 3339 UTC). It is the ordering key of the list -- the most recently photographed city leads -- and the city card's year span ends here. Equal to `takenAtFrom` when the place holds one member. 
  @BuiltValueField(wireName: r'takenAtTo')
  DateTime get takenAtTo;

  /// The place's cover candidates: its (up to) four most recent caller-visible live members, taken_at descending with the asset id as the tiebreak -- the wall's derived-cover discipline applied to city groups, newest first. 
  @BuiltValueField(wireName: r'representativeCoverAssetIds')
  BuiltList<String> get representativeCoverAssetIds;

  PlaceSummary._();

  factory PlaceSummary([void updates(PlaceSummaryBuilder b)]) = _$PlaceSummary;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(PlaceSummaryBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<PlaceSummary> get serializer => _$PlaceSummarySerializer();
}

class _$PlaceSummarySerializer implements PrimitiveSerializer<PlaceSummary> {
  @override
  final Iterable<Type> types = const [PlaceSummary, _$PlaceSummary];

  @override
  final String wireName = r'PlaceSummary';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    PlaceSummary object, {
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
    yield r'latitude';
    yield serializers.serialize(
      object.latitude,
      specifiedType: const FullType(double),
    );
    yield r'longitude';
    yield serializers.serialize(
      object.longitude,
      specifiedType: const FullType(double),
    );
    yield r'takenAtFrom';
    yield serializers.serialize(
      object.takenAtFrom,
      specifiedType: const FullType(DateTime),
    );
    yield r'takenAtTo';
    yield serializers.serialize(
      object.takenAtTo,
      specifiedType: const FullType(DateTime),
    );
    yield r'representativeCoverAssetIds';
    yield serializers.serialize(
      object.representativeCoverAssetIds,
      specifiedType: const FullType(BuiltList, [FullType(String)]),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    PlaceSummary object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required PlaceSummaryBuilder result,
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
        case r'latitude':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(double),
          ) as double;
          result.latitude = valueDes;
          break;
        case r'longitude':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(double),
          ) as double;
          result.longitude = valueDes;
          break;
        case r'takenAtFrom':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(DateTime),
          ) as DateTime;
          result.takenAtFrom = valueDes;
          break;
        case r'takenAtTo':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(DateTime),
          ) as DateTime;
          result.takenAtTo = valueDes;
          break;
        case r'representativeCoverAssetIds':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(BuiltList, [FullType(String)]),
          ) as BuiltList<String>;
          result.representativeCoverAssetIds.replace(valueDes);
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  PlaceSummary deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = PlaceSummaryBuilder();
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


