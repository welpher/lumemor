//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'map_config_response.g.dart';

/// The map's client-side configuration of `GET /map/config`: the light and dark tile sources the browser loads, each null when the instance has none. 
///
/// Properties:
/// * [tilejsonUrl] - The light tile source URL the browser loads directly from the tile host (a style document or a TileJSON descriptor); null means no source is configured and the map renders its stated absence. The dark theme falls back to this source when `darkTilejsonUrl` is null. The pin data never crosses the tile line: it comes from `GET /places`. 
/// * [darkTilejsonUrl] - The dark tile source URL, loaded by the dark theme instead of `tilejsonUrl`; null (or, equivalently, the instance not configuring one) means the dark theme reuses the light source. The light theme never loads it. 
@BuiltValue()
abstract class MapConfigResponse implements Built<MapConfigResponse, MapConfigResponseBuilder> {
  /// The light tile source URL the browser loads directly from the tile host (a style document or a TileJSON descriptor); null means no source is configured and the map renders its stated absence. The dark theme falls back to this source when `darkTilejsonUrl` is null. The pin data never crosses the tile line: it comes from `GET /places`. 
  @BuiltValueField(wireName: r'tilejsonUrl')
  String? get tilejsonUrl;

  /// The dark tile source URL, loaded by the dark theme instead of `tilejsonUrl`; null (or, equivalently, the instance not configuring one) means the dark theme reuses the light source. The light theme never loads it. 
  @BuiltValueField(wireName: r'darkTilejsonUrl')
  String? get darkTilejsonUrl;

  MapConfigResponse._();

  factory MapConfigResponse([void updates(MapConfigResponseBuilder b)]) = _$MapConfigResponse;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(MapConfigResponseBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<MapConfigResponse> get serializer => _$MapConfigResponseSerializer();
}

class _$MapConfigResponseSerializer implements PrimitiveSerializer<MapConfigResponse> {
  @override
  final Iterable<Type> types = const [MapConfigResponse, _$MapConfigResponse];

  @override
  final String wireName = r'MapConfigResponse';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    MapConfigResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'tilejsonUrl';
    yield object.tilejsonUrl == null ? null : serializers.serialize(
      object.tilejsonUrl,
      specifiedType: const FullType.nullable(String),
    );
    yield r'darkTilejsonUrl';
    yield object.darkTilejsonUrl == null ? null : serializers.serialize(
      object.darkTilejsonUrl,
      specifiedType: const FullType.nullable(String),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    MapConfigResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required MapConfigResponseBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'tilejsonUrl':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(String),
          ) as String?;
          if (valueDes == null) continue;
          result.tilejsonUrl = valueDes;
          break;
        case r'darkTilejsonUrl':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(String),
          ) as String?;
          if (valueDes == null) continue;
          result.darkTilejsonUrl = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  MapConfigResponse deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = MapConfigResponseBuilder();
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


