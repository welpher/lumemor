//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'update_check_settings_update_request.g.dart';

/// Payload of `PUT /admin/settings/update-check`.
///
/// Properties:
/// * [enabled] - The switch to save.
/// * [endpoint] - The release feed URL to save: an absolute `http` or `https` URL with a host. 
/// * [intervalHours] - The cadence to save, 1-168 hours.
@BuiltValue()
abstract class UpdateCheckSettingsUpdateRequest implements Built<UpdateCheckSettingsUpdateRequest, UpdateCheckSettingsUpdateRequestBuilder> {
  /// The switch to save.
  @BuiltValueField(wireName: r'enabled')
  bool get enabled;

  /// The release feed URL to save: an absolute `http` or `https` URL with a host. 
  @BuiltValueField(wireName: r'endpoint')
  String get endpoint;

  /// The cadence to save, 1-168 hours.
  @BuiltValueField(wireName: r'intervalHours')
  int get intervalHours;

  UpdateCheckSettingsUpdateRequest._();

  factory UpdateCheckSettingsUpdateRequest([void updates(UpdateCheckSettingsUpdateRequestBuilder b)]) = _$UpdateCheckSettingsUpdateRequest;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(UpdateCheckSettingsUpdateRequestBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<UpdateCheckSettingsUpdateRequest> get serializer => _$UpdateCheckSettingsUpdateRequestSerializer();
}

class _$UpdateCheckSettingsUpdateRequestSerializer implements PrimitiveSerializer<UpdateCheckSettingsUpdateRequest> {
  @override
  final Iterable<Type> types = const [UpdateCheckSettingsUpdateRequest, _$UpdateCheckSettingsUpdateRequest];

  @override
  final String wireName = r'UpdateCheckSettingsUpdateRequest';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    UpdateCheckSettingsUpdateRequest object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'enabled';
    yield serializers.serialize(
      object.enabled,
      specifiedType: const FullType(bool),
    );
    yield r'endpoint';
    yield serializers.serialize(
      object.endpoint,
      specifiedType: const FullType(String),
    );
    yield r'intervalHours';
    yield serializers.serialize(
      object.intervalHours,
      specifiedType: const FullType(int),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    UpdateCheckSettingsUpdateRequest object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required UpdateCheckSettingsUpdateRequestBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'enabled':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(bool),
          ) as bool;
          result.enabled = valueDes;
          break;
        case r'endpoint':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.endpoint = valueDes;
          break;
        case r'intervalHours':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(int),
          ) as int;
          result.intervalHours = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  UpdateCheckSettingsUpdateRequest deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = UpdateCheckSettingsUpdateRequestBuilder();
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


