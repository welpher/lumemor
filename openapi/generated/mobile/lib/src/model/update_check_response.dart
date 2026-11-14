//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:lumemor_api/src/model/update_check_status.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'update_check_response.g.dart';

/// The instance's release check state, served by `GET /server/update-check` and the manual `POST`. 
///
/// Properties:
/// * [enabled] - Whether a check runs at all: the settings switch AND the deployment's own switch, both closed for `true`. 
/// * [status] 
/// * [currentVersion] - The running server version -- the build's version string, verbatim (development builds report `dev`, which the comparison cannot rank; such an instance stays `unknown`). 
/// * [latestVersion] - The newest tag the feed named, verbatim including its `v` prefix, or null when no probe has succeeded yet. 
/// * [checkedAt] - When the last probe completed, successful or not; null before the first probe. The settings face renders it as the quiet \"last checked\" line when a probe failed. 
@BuiltValue()
abstract class UpdateCheckResponse implements Built<UpdateCheckResponse, UpdateCheckResponseBuilder> {
  /// Whether a check runs at all: the settings switch AND the deployment's own switch, both closed for `true`. 
  @BuiltValueField(wireName: r'enabled')
  bool get enabled;

  @BuiltValueField(wireName: r'status')
  UpdateCheckStatus get status;
  // enum statusEnum {  disabled,  unknown,  up_to_date,  update_available,  };

  /// The running server version -- the build's version string, verbatim (development builds report `dev`, which the comparison cannot rank; such an instance stays `unknown`). 
  @BuiltValueField(wireName: r'currentVersion')
  String get currentVersion;

  /// The newest tag the feed named, verbatim including its `v` prefix, or null when no probe has succeeded yet. 
  @BuiltValueField(wireName: r'latestVersion')
  String? get latestVersion;

  /// When the last probe completed, successful or not; null before the first probe. The settings face renders it as the quiet \"last checked\" line when a probe failed. 
  @BuiltValueField(wireName: r'checkedAt')
  DateTime? get checkedAt;

  UpdateCheckResponse._();

  factory UpdateCheckResponse([void updates(UpdateCheckResponseBuilder b)]) = _$UpdateCheckResponse;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(UpdateCheckResponseBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<UpdateCheckResponse> get serializer => _$UpdateCheckResponseSerializer();
}

class _$UpdateCheckResponseSerializer implements PrimitiveSerializer<UpdateCheckResponse> {
  @override
  final Iterable<Type> types = const [UpdateCheckResponse, _$UpdateCheckResponse];

  @override
  final String wireName = r'UpdateCheckResponse';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    UpdateCheckResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'enabled';
    yield serializers.serialize(
      object.enabled,
      specifiedType: const FullType(bool),
    );
    yield r'status';
    yield serializers.serialize(
      object.status,
      specifiedType: const FullType(UpdateCheckStatus),
    );
    yield r'currentVersion';
    yield serializers.serialize(
      object.currentVersion,
      specifiedType: const FullType(String),
    );
    yield r'latestVersion';
    yield object.latestVersion == null ? null : serializers.serialize(
      object.latestVersion,
      specifiedType: const FullType.nullable(String),
    );
    yield r'checkedAt';
    yield object.checkedAt == null ? null : serializers.serialize(
      object.checkedAt,
      specifiedType: const FullType.nullable(DateTime),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    UpdateCheckResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required UpdateCheckResponseBuilder result,
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
        case r'status':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(UpdateCheckStatus),
          ) as UpdateCheckStatus;
          result.status = valueDes;
          break;
        case r'currentVersion':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.currentVersion = valueDes;
          break;
        case r'latestVersion':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(String),
          ) as String?;
          if (valueDes == null) continue;
          result.latestVersion = valueDes;
          break;
        case r'checkedAt':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(DateTime),
          ) as DateTime?;
          if (valueDes == null) continue;
          result.checkedAt = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  UpdateCheckResponse deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = UpdateCheckResponseBuilder();
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


