//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'update_check_settings_response.g.dart';

/// The release check's settings of `GET /admin/settings/update-check` and the successful `PUT`. 
///
/// Properties:
/// * [enabled] - The switch in force at the settings layer. `false` stops every scheduled and manual check; the cached answer is kept, so switching it back on resumes from the same state. 
/// * [endpoint] - The release feed URL the server probes: a JSON document whose newest tag rides `tag` (or `tag_name`). The server fetches it with a bounded timeout, through the environment's forward proxy, refusing internal addresses and never following a redirect. 
/// * [intervalHours] - The cadence in hours, 1-168: the server probes at most once per interval and caches the answer until the next one. 
@BuiltValue()
abstract class UpdateCheckSettingsResponse implements Built<UpdateCheckSettingsResponse, UpdateCheckSettingsResponseBuilder> {
  /// The switch in force at the settings layer. `false` stops every scheduled and manual check; the cached answer is kept, so switching it back on resumes from the same state. 
  @BuiltValueField(wireName: r'enabled')
  bool get enabled;

  /// The release feed URL the server probes: a JSON document whose newest tag rides `tag` (or `tag_name`). The server fetches it with a bounded timeout, through the environment's forward proxy, refusing internal addresses and never following a redirect. 
  @BuiltValueField(wireName: r'endpoint')
  String get endpoint;

  /// The cadence in hours, 1-168: the server probes at most once per interval and caches the answer until the next one. 
  @BuiltValueField(wireName: r'intervalHours')
  int get intervalHours;

  UpdateCheckSettingsResponse._();

  factory UpdateCheckSettingsResponse([void updates(UpdateCheckSettingsResponseBuilder b)]) = _$UpdateCheckSettingsResponse;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(UpdateCheckSettingsResponseBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<UpdateCheckSettingsResponse> get serializer => _$UpdateCheckSettingsResponseSerializer();
}

class _$UpdateCheckSettingsResponseSerializer implements PrimitiveSerializer<UpdateCheckSettingsResponse> {
  @override
  final Iterable<Type> types = const [UpdateCheckSettingsResponse, _$UpdateCheckSettingsResponse];

  @override
  final String wireName = r'UpdateCheckSettingsResponse';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    UpdateCheckSettingsResponse object, {
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
    UpdateCheckSettingsResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required UpdateCheckSettingsResponseBuilder result,
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
  UpdateCheckSettingsResponse deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = UpdateCheckSettingsResponseBuilder();
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


