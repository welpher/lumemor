//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_collection/built_collection.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'mount_roots_settings_update_request.g.dart';

/// Payload of `PUT /admin/settings/mount-roots`: the COMPLETE replacement allowlist. The field must ride the body -- an explicit empty array closes directory album creation, while an absent field is a client bug the 400 names. 
///
/// Properties:
/// * [roots] - Every mount root of the new allowlist, at most 32 entries. Each must be an absolute path within the 4,096-byte bound; the server cleans each value and collapses duplicates after the clean. Existence is not required -- a root may be configured before the volume is mounted. 
@BuiltValue()
abstract class MountRootsSettingsUpdateRequest implements Built<MountRootsSettingsUpdateRequest, MountRootsSettingsUpdateRequestBuilder> {
  /// Every mount root of the new allowlist, at most 32 entries. Each must be an absolute path within the 4,096-byte bound; the server cleans each value and collapses duplicates after the clean. Existence is not required -- a root may be configured before the volume is mounted. 
  @BuiltValueField(wireName: r'roots')
  BuiltList<String> get roots;

  MountRootsSettingsUpdateRequest._();

  factory MountRootsSettingsUpdateRequest([void updates(MountRootsSettingsUpdateRequestBuilder b)]) = _$MountRootsSettingsUpdateRequest;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(MountRootsSettingsUpdateRequestBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<MountRootsSettingsUpdateRequest> get serializer => _$MountRootsSettingsUpdateRequestSerializer();
}

class _$MountRootsSettingsUpdateRequestSerializer implements PrimitiveSerializer<MountRootsSettingsUpdateRequest> {
  @override
  final Iterable<Type> types = const [MountRootsSettingsUpdateRequest, _$MountRootsSettingsUpdateRequest];

  @override
  final String wireName = r'MountRootsSettingsUpdateRequest';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    MountRootsSettingsUpdateRequest object, {
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
    MountRootsSettingsUpdateRequest object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required MountRootsSettingsUpdateRequestBuilder result,
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
  MountRootsSettingsUpdateRequest deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = MountRootsSettingsUpdateRequestBuilder();
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


