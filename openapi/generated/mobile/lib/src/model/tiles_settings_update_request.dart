//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'tiles_settings_update_request.g.dart';

/// Payload of `PUT /admin/settings/tiles`: both styles, one document. The save replaces the pair wholesale. 
///
/// Properties:
/// * [tilejsonUrl] - The light tile source URL to save, or the empty string to clear it. A non-empty value must be an absolute `http` or `https` URL; a LAN address is legitimate (the browser loads the source, not the server). 
/// * [darkTilejsonUrl] - The dark tile source URL to save. Omitting it or sending the empty string clears it (the dark theme then reuses the light source). A non-empty value follows the light field's rule: an absolute `http` or `https` URL, a LAN address included. 
/// * [gcj02] - The light source's coordinate system: `true` for a GCJ-02 basemap (Amap, Tencent), `false` or omitted for the standard WGS-84. Omitting it saves the false default, so a client unaware of the flag keeps behaving exactly as before. 
/// * [darkGcj02] - The dark source's coordinate system, on the same rule as `gcj02`: `true` for a GCJ-02 basemap, `false` or omitted for WGS-84. Ignored while the dark source is empty (the light source then governs both themes). 
@BuiltValue()
abstract class TilesSettingsUpdateRequest implements Built<TilesSettingsUpdateRequest, TilesSettingsUpdateRequestBuilder> {
  /// The light tile source URL to save, or the empty string to clear it. A non-empty value must be an absolute `http` or `https` URL; a LAN address is legitimate (the browser loads the source, not the server). 
  @BuiltValueField(wireName: r'tilejsonUrl')
  String get tilejsonUrl;

  /// The dark tile source URL to save. Omitting it or sending the empty string clears it (the dark theme then reuses the light source). A non-empty value follows the light field's rule: an absolute `http` or `https` URL, a LAN address included. 
  @BuiltValueField(wireName: r'darkTilejsonUrl')
  String? get darkTilejsonUrl;

  /// The light source's coordinate system: `true` for a GCJ-02 basemap (Amap, Tencent), `false` or omitted for the standard WGS-84. Omitting it saves the false default, so a client unaware of the flag keeps behaving exactly as before. 
  @BuiltValueField(wireName: r'gcj02')
  bool? get gcj02;

  /// The dark source's coordinate system, on the same rule as `gcj02`: `true` for a GCJ-02 basemap, `false` or omitted for WGS-84. Ignored while the dark source is empty (the light source then governs both themes). 
  @BuiltValueField(wireName: r'darkGcj02')
  bool? get darkGcj02;

  TilesSettingsUpdateRequest._();

  factory TilesSettingsUpdateRequest([void updates(TilesSettingsUpdateRequestBuilder b)]) = _$TilesSettingsUpdateRequest;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(TilesSettingsUpdateRequestBuilder b) => b
      ..gcj02 = false
      ..darkGcj02 = false;

  @BuiltValueSerializer(custom: true)
  static Serializer<TilesSettingsUpdateRequest> get serializer => _$TilesSettingsUpdateRequestSerializer();
}

class _$TilesSettingsUpdateRequestSerializer implements PrimitiveSerializer<TilesSettingsUpdateRequest> {
  @override
  final Iterable<Type> types = const [TilesSettingsUpdateRequest, _$TilesSettingsUpdateRequest];

  @override
  final String wireName = r'TilesSettingsUpdateRequest';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    TilesSettingsUpdateRequest object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'tilejsonUrl';
    yield serializers.serialize(
      object.tilejsonUrl,
      specifiedType: const FullType(String),
    );
    if (object.darkTilejsonUrl != null) {
      yield r'darkTilejsonUrl';
      yield serializers.serialize(
        object.darkTilejsonUrl,
        specifiedType: const FullType(String),
      );
    }
    if (object.gcj02 != null) {
      yield r'gcj02';
      yield serializers.serialize(
        object.gcj02,
        specifiedType: const FullType(bool),
      );
    }
    if (object.darkGcj02 != null) {
      yield r'darkGcj02';
      yield serializers.serialize(
        object.darkGcj02,
        specifiedType: const FullType(bool),
      );
    }
  }

  @override
  Object serialize(
    Serializers serializers,
    TilesSettingsUpdateRequest object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required TilesSettingsUpdateRequestBuilder result,
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
            specifiedType: const FullType.nullable(String),
          ) as String?;
          if (valueDes == null) continue;
          result.darkTilejsonUrl = valueDes;
          break;
        case r'gcj02':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(bool),
          ) as bool?;
          if (valueDes == null) continue;
          result.gcj02 = valueDes;
          break;
        case r'darkGcj02':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(bool),
          ) as bool?;
          if (valueDes == null) continue;
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
  TilesSettingsUpdateRequest deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = TilesSettingsUpdateRequestBuilder();
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


