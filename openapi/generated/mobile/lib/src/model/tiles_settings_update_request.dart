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
@BuiltValue()
abstract class TilesSettingsUpdateRequest implements Built<TilesSettingsUpdateRequest, TilesSettingsUpdateRequestBuilder> {
  /// The light tile source URL to save, or the empty string to clear it. A non-empty value must be an absolute `http` or `https` URL; a LAN address is legitimate (the browser loads the source, not the server). 
  @BuiltValueField(wireName: r'tilejsonUrl')
  String get tilejsonUrl;

  /// The dark tile source URL to save. Omitting it or sending the empty string clears it (the dark theme then reuses the light source). A non-empty value follows the light field's rule: an absolute `http` or `https` URL, a LAN address included. 
  @BuiltValueField(wireName: r'darkTilejsonUrl')
  String? get darkTilejsonUrl;

  TilesSettingsUpdateRequest._();

  factory TilesSettingsUpdateRequest([void updates(TilesSettingsUpdateRequestBuilder b)]) = _$TilesSettingsUpdateRequest;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(TilesSettingsUpdateRequestBuilder b) => b;

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


