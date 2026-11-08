//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'trash_settings_update_request.g.dart';

/// Payload of `PUT /admin/settings/trash`.
///
/// Properties:
/// * [retentionDays] - The retention to save, in whole days, 0 to 365.
@BuiltValue()
abstract class TrashSettingsUpdateRequest implements Built<TrashSettingsUpdateRequest, TrashSettingsUpdateRequestBuilder> {
  /// The retention to save, in whole days, 0 to 365.
  @BuiltValueField(wireName: r'retentionDays')
  int get retentionDays;

  TrashSettingsUpdateRequest._();

  factory TrashSettingsUpdateRequest([void updates(TrashSettingsUpdateRequestBuilder b)]) = _$TrashSettingsUpdateRequest;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(TrashSettingsUpdateRequestBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<TrashSettingsUpdateRequest> get serializer => _$TrashSettingsUpdateRequestSerializer();
}

class _$TrashSettingsUpdateRequestSerializer implements PrimitiveSerializer<TrashSettingsUpdateRequest> {
  @override
  final Iterable<Type> types = const [TrashSettingsUpdateRequest, _$TrashSettingsUpdateRequest];

  @override
  final String wireName = r'TrashSettingsUpdateRequest';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    TrashSettingsUpdateRequest object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'retentionDays';
    yield serializers.serialize(
      object.retentionDays,
      specifiedType: const FullType(int),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    TrashSettingsUpdateRequest object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required TrashSettingsUpdateRequestBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'retentionDays':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(int),
          ) as int;
          result.retentionDays = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  TrashSettingsUpdateRequest deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = TrashSettingsUpdateRequestBuilder();
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


