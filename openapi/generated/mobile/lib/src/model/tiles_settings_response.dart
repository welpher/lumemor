//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'tiles_settings_response.g.dart';

/// The map tile sources of `GET /admin/settings/tiles` and the successful `PUT`: the light style plus the optional dark one, each empty when unconfigured. 
///
/// Properties:
/// * [tilejsonUrl] - The light tile source URL: a style document (vector) or a TileJSON descriptor (raster), loaded by the browser in the light theme and in the dark theme when no dark style is configured. The empty string is the shipped state: no source, and the map draws its pins over the stated absence of a base layer. 
/// * [darkTilejsonUrl] - The dark tile source URL, same shapes as the light one. The empty string means the dark theme reuses `tilejsonUrl` -- one source, one look in both themes. A configured value is loaded by the dark theme only. 
/// * [gcj02] - The light source's coordinate system: `true` when its basemap draws in GCJ-02 (the offset datum Amap and Tencent use inside China), `false` for the standard WGS-84 every other source draws in. The flag is a property of the source, not of the library: the stored photo coordinates stay WGS-84 and the map converts only at the display and pick boundaries. 
/// * [darkGcj02] - The dark source's coordinate system, on the same rule as `gcj02`. An empty dark source reuses the light one, and the light source's flag then governs the dark theme too. 
@BuiltValue()
abstract class TilesSettingsResponse implements Built<TilesSettingsResponse, TilesSettingsResponseBuilder> {
  /// The light tile source URL: a style document (vector) or a TileJSON descriptor (raster), loaded by the browser in the light theme and in the dark theme when no dark style is configured. The empty string is the shipped state: no source, and the map draws its pins over the stated absence of a base layer. 
  @BuiltValueField(wireName: r'tilejsonUrl')
  String get tilejsonUrl;

  /// The dark tile source URL, same shapes as the light one. The empty string means the dark theme reuses `tilejsonUrl` -- one source, one look in both themes. A configured value is loaded by the dark theme only. 
  @BuiltValueField(wireName: r'darkTilejsonUrl')
  String get darkTilejsonUrl;

  /// The light source's coordinate system: `true` when its basemap draws in GCJ-02 (the offset datum Amap and Tencent use inside China), `false` for the standard WGS-84 every other source draws in. The flag is a property of the source, not of the library: the stored photo coordinates stay WGS-84 and the map converts only at the display and pick boundaries. 
  @BuiltValueField(wireName: r'gcj02')
  bool get gcj02;

  /// The dark source's coordinate system, on the same rule as `gcj02`. An empty dark source reuses the light one, and the light source's flag then governs the dark theme too. 
  @BuiltValueField(wireName: r'darkGcj02')
  bool get darkGcj02;

  TilesSettingsResponse._();

  factory TilesSettingsResponse([void updates(TilesSettingsResponseBuilder b)]) = _$TilesSettingsResponse;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(TilesSettingsResponseBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<TilesSettingsResponse> get serializer => _$TilesSettingsResponseSerializer();
}

class _$TilesSettingsResponseSerializer implements PrimitiveSerializer<TilesSettingsResponse> {
  @override
  final Iterable<Type> types = const [TilesSettingsResponse, _$TilesSettingsResponse];

  @override
  final String wireName = r'TilesSettingsResponse';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    TilesSettingsResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'tilejsonUrl';
    yield serializers.serialize(
      object.tilejsonUrl,
      specifiedType: const FullType(String),
    );
    yield r'darkTilejsonUrl';
    yield serializers.serialize(
      object.darkTilejsonUrl,
      specifiedType: const FullType(String),
    );
    yield r'gcj02';
    yield serializers.serialize(
      object.gcj02,
      specifiedType: const FullType(bool),
    );
    yield r'darkGcj02';
    yield serializers.serialize(
      object.darkGcj02,
      specifiedType: const FullType(bool),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    TilesSettingsResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required TilesSettingsResponseBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'tilejsonUrl':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.tilejsonUrl = valueDes;
          break;
        case r'darkTilejsonUrl':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.darkTilejsonUrl = valueDes;
          break;
        case r'gcj02':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(bool),
          ) as bool;
          result.gcj02 = valueDes;
          break;
        case r'darkGcj02':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(bool),
          ) as bool;
          result.darkGcj02 = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  TilesSettingsResponse deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = TilesSettingsResponseBuilder();
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


