//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'trash_settings_response.g.dart';

/// The trash settings of `GET /admin/settings/trash` and the successful `PUT`: the instance's trash retention in whole days. 
///
/// Properties:
/// * [retentionDays] - The retention window of a soft-deleted asset in whole days, 0 to 365. A fresh instance (or a corrupt row) answers the default 30; 0 means the next expiry run purges everything trashed. 
@BuiltValue()
abstract class TrashSettingsResponse implements Built<TrashSettingsResponse, TrashSettingsResponseBuilder> {
  /// The retention window of a soft-deleted asset in whole days, 0 to 365. A fresh instance (or a corrupt row) answers the default 30; 0 means the next expiry run purges everything trashed. 
  @BuiltValueField(wireName: r'retentionDays')
  int get retentionDays;

  TrashSettingsResponse._();

  factory TrashSettingsResponse([void updates(TrashSettingsResponseBuilder b)]) = _$TrashSettingsResponse;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(TrashSettingsResponseBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<TrashSettingsResponse> get serializer => _$TrashSettingsResponseSerializer();
}

class _$TrashSettingsResponseSerializer implements PrimitiveSerializer<TrashSettingsResponse> {
  @override
  final Iterable<Type> types = const [TrashSettingsResponse, _$TrashSettingsResponse];

  @override
  final String wireName = r'TrashSettingsResponse';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    TrashSettingsResponse object, {
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
    TrashSettingsResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required TrashSettingsResponseBuilder result,
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
  TrashSettingsResponse deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = TrashSettingsResponseBuilder();
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


