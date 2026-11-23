//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:lumemor_api/src/model/tag_facet_count.dart';
import 'package:lumemor_api/src/model/camera_facet_count.dart';
import 'package:built_collection/built_collection.dart';
import 'package:lumemor_api/src/model/album_facet_count.dart';
import 'package:lumemor_api/src/model/year_facet_count.dart';
import 'package:lumemor_api/src/model/person_facet_count.dart';
import 'package:lumemor_api/src/model/place_facet_count.dart';
import 'package:lumemor_api/src/model/kind_facet_count.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'search_facets.g.dart';

/// The filter bar's counts over the current filtered set minus each dimension: a facet entry's `assetCount` answers \"how many results would I have if I added this value ON TOP of the current query and filters\" (the dimension's own filter parameters excluded). Every facet is caller-scoped exactly like the surface it summarizes: places count visible assets, tags and people are the caller's own, albums are the ones the caller can see. The lists are ordered by `assetCount` descending with a deterministic per-dimension tiebreak; they are not paginated (bounded by the family instance's vocabulary sizes, the tags listing's own shape). 
///
/// Properties:
/// * [people] - The caller's named people: one entry per labeled face cluster with a face on the current set. 
/// * [places] - The resolved places of the current set with the locale display name (`locale` parameter, the `GET /places` join). 
/// * [years] - The taken-at years of the current set, newest first: the time-range facet (the `from`/`to` filters are the dimension the year counts exclude). 
/// * [cameras] - The EXIF camera makes/models of the current set. The `id` is the round-trip filter value (`make|model`); `make` and `model` are the display halves, null exactly when the EXIF row carries no value there. 
/// * [kinds] - The media families of the current set.
/// * [albums] - The caller-visible albums (own and shared, never the sync albums -- the timeline is their face) holding members of the current set. 
/// * [tags] - The CALLER's tags with members in the current set (another account's tags never surface). 
@BuiltValue()
abstract class SearchFacets implements Built<SearchFacets, SearchFacetsBuilder> {
  /// The caller's named people: one entry per labeled face cluster with a face on the current set. 
  @BuiltValueField(wireName: r'people')
  BuiltList<PersonFacetCount> get people;

  /// The resolved places of the current set with the locale display name (`locale` parameter, the `GET /places` join). 
  @BuiltValueField(wireName: r'places')
  BuiltList<PlaceFacetCount> get places;

  /// The taken-at years of the current set, newest first: the time-range facet (the `from`/`to` filters are the dimension the year counts exclude). 
  @BuiltValueField(wireName: r'years')
  BuiltList<YearFacetCount> get years;

  /// The EXIF camera makes/models of the current set. The `id` is the round-trip filter value (`make|model`); `make` and `model` are the display halves, null exactly when the EXIF row carries no value there. 
  @BuiltValueField(wireName: r'cameras')
  BuiltList<CameraFacetCount> get cameras;

  /// The media families of the current set.
  @BuiltValueField(wireName: r'kinds')
  BuiltList<KindFacetCount> get kinds;

  /// The caller-visible albums (own and shared, never the sync albums -- the timeline is their face) holding members of the current set. 
  @BuiltValueField(wireName: r'albums')
  BuiltList<AlbumFacetCount> get albums;

  /// The CALLER's tags with members in the current set (another account's tags never surface). 
  @BuiltValueField(wireName: r'tags')
  BuiltList<TagFacetCount> get tags;

  SearchFacets._();

  factory SearchFacets([void updates(SearchFacetsBuilder b)]) = _$SearchFacets;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(SearchFacetsBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<SearchFacets> get serializer => _$SearchFacetsSerializer();
}

class _$SearchFacetsSerializer implements PrimitiveSerializer<SearchFacets> {
  @override
  final Iterable<Type> types = const [SearchFacets, _$SearchFacets];

  @override
  final String wireName = r'SearchFacets';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    SearchFacets object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'people';
    yield serializers.serialize(
      object.people,
      specifiedType: const FullType(BuiltList, [FullType(PersonFacetCount)]),
    );
    yield r'places';
    yield serializers.serialize(
      object.places,
      specifiedType: const FullType(BuiltList, [FullType(PlaceFacetCount)]),
    );
    yield r'years';
    yield serializers.serialize(
      object.years,
      specifiedType: const FullType(BuiltList, [FullType(YearFacetCount)]),
    );
    yield r'cameras';
    yield serializers.serialize(
      object.cameras,
      specifiedType: const FullType(BuiltList, [FullType(CameraFacetCount)]),
    );
    yield r'kinds';
    yield serializers.serialize(
      object.kinds,
      specifiedType: const FullType(BuiltList, [FullType(KindFacetCount)]),
    );
    yield r'albums';
    yield serializers.serialize(
      object.albums,
      specifiedType: const FullType(BuiltList, [FullType(AlbumFacetCount)]),
    );
    yield r'tags';
    yield serializers.serialize(
      object.tags,
      specifiedType: const FullType(BuiltList, [FullType(TagFacetCount)]),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    SearchFacets object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required SearchFacetsBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'people':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(BuiltList, [FullType(PersonFacetCount)]),
          ) as BuiltList<PersonFacetCount>;
          result.people.replace(valueDes);
          break;
        case r'places':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(BuiltList, [FullType(PlaceFacetCount)]),
          ) as BuiltList<PlaceFacetCount>;
          result.places.replace(valueDes);
          break;
        case r'years':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(BuiltList, [FullType(YearFacetCount)]),
          ) as BuiltList<YearFacetCount>;
          result.years.replace(valueDes);
          break;
        case r'cameras':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(BuiltList, [FullType(CameraFacetCount)]),
          ) as BuiltList<CameraFacetCount>;
          result.cameras.replace(valueDes);
          break;
        case r'kinds':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(BuiltList, [FullType(KindFacetCount)]),
          ) as BuiltList<KindFacetCount>;
          result.kinds.replace(valueDes);
          break;
        case r'albums':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(BuiltList, [FullType(AlbumFacetCount)]),
          ) as BuiltList<AlbumFacetCount>;
          result.albums.replace(valueDes);
          break;
        case r'tags':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(BuiltList, [FullType(TagFacetCount)]),
          ) as BuiltList<TagFacetCount>;
          result.tags.replace(valueDes);
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  SearchFacets deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = SearchFacetsBuilder();
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


