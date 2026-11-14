//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'tiles_settings_response.g.dart';

/// The map tile source of `GET /admin/settings/tiles` and the successful `PUT`: one URL, empty when the instance has none. 
///
/// Properties:
/// * [tilejsonUrl] - The tile source URL: a style document (vector) or a TileJSON descriptor (raster), loaded by the browser. The empty string is the shipped state: no source, and the map draws its pins over the stated absence of a base layer. 
@BuiltValue()
abstract class TilesSettingsResponse implements Built<TilesSettingsResponse, TilesSettingsResponseBuilder> {
  /// The tile source URL: a style document (vector) or a TileJSON descriptor (raster), loaded by the browser. The empty string is the shipped state: no source, and the map draws its pins over the stated absence of a base layer. 
  @BuiltValueField(wireName: r'tilejsonUrl')
  String get tilejsonUrl;

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


