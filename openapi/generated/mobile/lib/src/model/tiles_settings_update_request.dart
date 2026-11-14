//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'tiles_settings_update_request.g.dart';

/// Payload of `PUT /admin/settings/tiles`.
///
/// Properties:
/// * [tilejsonUrl] - The tile source URL to save, or the empty string to clear it. A non-empty value must be an absolute `http` or `https` URL; a LAN address is legitimate (the browser loads the source, not the server). 
@BuiltValue()
abstract class TilesSettingsUpdateRequest implements Built<TilesSettingsUpdateRequest, TilesSettingsUpdateRequestBuilder> {
  /// The tile source URL to save, or the empty string to clear it. A non-empty value must be an absolute `http` or `https` URL; a LAN address is legitimate (the browser loads the source, not the server). 
  @BuiltValueField(wireName: r'tilejsonUrl')
  String get tilejsonUrl;

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


