//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_collection/built_collection.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'places_database_response.g.dart';

/// The loaded place database: its data-version marker, the display-name locale set it carries, and the two re-resolution scopes' sizes (the dialog's estimates). 
///
/// Properties:
/// * [version] - The data-version marker the startup import recorded -- the sha256 digest of the exact loaded bytes. An empty string means the instance carries no bake (nothing resolves). A 64-character value is a content digest, not a release date: the card displays its first eight characters. 
/// * [languages] - The display-name locales the bake carries, in the product's language-set order. The places read's `locale` parameter accepts exactly this set. 
/// * [gpsAssets] - The full re-resolution scope: every asset whose exif row carries GPS coordinates, whatever its lifecycle state. A row with a manual location is excluded -- the walk leaves those rows to the owner's own decision, so the count and the walk cover the same rows. 
/// * [missingPlaces] - The fill-in scope: the GPS-bearing assets whose place is absent (the subset a `missing` run walks), with the same manual-location exclusion as `gpsAssets`. 
@BuiltValue()
abstract class PlacesDatabaseResponse implements Built<PlacesDatabaseResponse, PlacesDatabaseResponseBuilder> {
  /// The data-version marker the startup import recorded -- the sha256 digest of the exact loaded bytes. An empty string means the instance carries no bake (nothing resolves). A 64-character value is a content digest, not a release date: the card displays its first eight characters. 
  @BuiltValueField(wireName: r'version')
  String get version;

  /// The display-name locales the bake carries, in the product's language-set order. The places read's `locale` parameter accepts exactly this set. 
  @BuiltValueField(wireName: r'languages')
  BuiltList<String> get languages;

  /// The full re-resolution scope: every asset whose exif row carries GPS coordinates, whatever its lifecycle state. A row with a manual location is excluded -- the walk leaves those rows to the owner's own decision, so the count and the walk cover the same rows. 
  @BuiltValueField(wireName: r'gpsAssets')
  int get gpsAssets;

  /// The fill-in scope: the GPS-bearing assets whose place is absent (the subset a `missing` run walks), with the same manual-location exclusion as `gpsAssets`. 
  @BuiltValueField(wireName: r'missingPlaces')
  int get missingPlaces;

  PlacesDatabaseResponse._();

  factory PlacesDatabaseResponse([void updates(PlacesDatabaseResponseBuilder b)]) = _$PlacesDatabaseResponse;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(PlacesDatabaseResponseBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<PlacesDatabaseResponse> get serializer => _$PlacesDatabaseResponseSerializer();
}

class _$PlacesDatabaseResponseSerializer implements PrimitiveSerializer<PlacesDatabaseResponse> {
  @override
  final Iterable<Type> types = const [PlacesDatabaseResponse, _$PlacesDatabaseResponse];

  @override
  final String wireName = r'PlacesDatabaseResponse';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    PlacesDatabaseResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'version';
    yield serializers.serialize(
      object.version,
      specifiedType: const FullType(String),
    );
    yield r'languages';
    yield serializers.serialize(
      object.languages,
      specifiedType: const FullType(BuiltList, [FullType(String)]),
    );
    yield r'gpsAssets';
    yield serializers.serialize(
      object.gpsAssets,
      specifiedType: const FullType(int),
    );
    yield r'missingPlaces';
    yield serializers.serialize(
      object.missingPlaces,
      specifiedType: const FullType(int),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    PlacesDatabaseResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required PlacesDatabaseResponseBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'version':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.version = valueDes;
          break;
        case r'languages':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(BuiltList, [FullType(String)]),
          ) as BuiltList<String>;
          result.languages.replace(valueDes);
          break;
        case r'gpsAssets':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(int),
          ) as int;
          result.gpsAssets = valueDes;
          break;
        case r'missingPlaces':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(int),
          ) as int;
          result.missingPlaces = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  PlacesDatabaseResponse deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = PlacesDatabaseResponseBuilder();
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


