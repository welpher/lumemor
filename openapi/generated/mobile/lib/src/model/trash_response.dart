//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_collection/built_collection.dart';
import 'package:lumemor_api/src/model/trash_item.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'trash_response.g.dart';

/// The trash view: the caller's trashed assets, newest deletion first. The list is bounded (the 200 most recent deletions) and unpaginated -- the trash is a recovery view for recent deletes, and items age out through expiry on their own. 
///
/// Properties:
/// * [items] - The trashed assets of the caller, ordered by `deletedAt` descending. 
/// * [retentionDays] - The instance's trash retention window in whole days, the same setting the listing's `remainingDays` countdowns were computed against (`trash_retention_days`, default 30). The trash page's note renders it; adjusting the setting stays the administrator's surface, but the number itself is visible to every signed-in member. A corrupt setting row answers the default 30 -- the same degradation the countdowns follow. 
@BuiltValue()
abstract class TrashResponse implements Built<TrashResponse, TrashResponseBuilder> {
  /// The trashed assets of the caller, ordered by `deletedAt` descending. 
  @BuiltValueField(wireName: r'items')
  BuiltList<TrashItem> get items;

  /// The instance's trash retention window in whole days, the same setting the listing's `remainingDays` countdowns were computed against (`trash_retention_days`, default 30). The trash page's note renders it; adjusting the setting stays the administrator's surface, but the number itself is visible to every signed-in member. A corrupt setting row answers the default 30 -- the same degradation the countdowns follow. 
  @BuiltValueField(wireName: r'retentionDays')
  int get retentionDays;

  TrashResponse._();

  factory TrashResponse([void updates(TrashResponseBuilder b)]) = _$TrashResponse;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(TrashResponseBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<TrashResponse> get serializer => _$TrashResponseSerializer();
}

class _$TrashResponseSerializer implements PrimitiveSerializer<TrashResponse> {
  @override
  final Iterable<Type> types = const [TrashResponse, _$TrashResponse];

  @override
  final String wireName = r'TrashResponse';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    TrashResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'items';
    yield serializers.serialize(
      object.items,
      specifiedType: const FullType(BuiltList, [FullType(TrashItem)]),
    );
    yield r'retentionDays';
    yield serializers.serialize(
      object.retentionDays,
      specifiedType: const FullType(int),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    TrashResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required TrashResponseBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'items':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(BuiltList, [FullType(TrashItem)]),
          ) as BuiltList<TrashItem>;
          result.items.replace(valueDes);
          break;
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
  TrashResponse deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = TrashResponseBuilder();
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


