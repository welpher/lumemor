//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_collection/built_collection.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'mount_roots_settings_response.g.dart';

/// The mount-root allowlist answer of `GET /admin/settings/ mount-roots` and `PUT /admin/settings/mount-roots`: the stored list verbatim (the PUT's answer carries the cleaned, deduplicated value in first-occurrence order). The empty list is the default closed posture, never an error. 
///
/// Properties:
/// * [roots] - The configured mount roots: absolute directory paths, at most 32, stored as written. A root's live usability (exists, is a directory, outside the managed tree) is the completion and creation domains' concern, not this list's -- the list is the honest record of the configuration. 
@BuiltValue()
abstract class MountRootsSettingsResponse implements Built<MountRootsSettingsResponse, MountRootsSettingsResponseBuilder> {
  /// The configured mount roots: absolute directory paths, at most 32, stored as written. A root's live usability (exists, is a directory, outside the managed tree) is the completion and creation domains' concern, not this list's -- the list is the honest record of the configuration. 
  @BuiltValueField(wireName: r'roots')
  BuiltList<String> get roots;

  MountRootsSettingsResponse._();

  factory MountRootsSettingsResponse([void updates(MountRootsSettingsResponseBuilder b)]) = _$MountRootsSettingsResponse;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(MountRootsSettingsResponseBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<MountRootsSettingsResponse> get serializer => _$MountRootsSettingsResponseSerializer();
}

class _$MountRootsSettingsResponseSerializer implements PrimitiveSerializer<MountRootsSettingsResponse> {
  @override
  final Iterable<Type> types = const [MountRootsSettingsResponse, _$MountRootsSettingsResponse];

  @override
  final String wireName = r'MountRootsSettingsResponse';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    MountRootsSettingsResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'roots';
    yield serializers.serialize(
      object.roots,
      specifiedType: const FullType(BuiltList, [FullType(String)]),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    MountRootsSettingsResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required MountRootsSettingsResponseBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'roots':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(BuiltList, [FullType(String)]),
          ) as BuiltList<String>;
          result.roots.replace(valueDes);
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  MountRootsSettingsResponse deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = MountRootsSettingsResponseBuilder();
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


