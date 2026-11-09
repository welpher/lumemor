//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_collection/built_collection.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'album_audience_skip.g.dart';

/// One requested audience entry the write could not grant: the id and the honest reason. The reason is a machine value the client maps onto its own phrasing; it never probes the account table further. 
///
/// Properties:
/// * [userId] - The requested id that stayed ungranted.
/// * [reason] - Why the id was skipped: `owner` -- the album's own owner, whose visibility is the ownership row, never a share row; `invalid_account` -- an unknown or disabled account (the same reason for both, so user ids cannot be probed). 
@BuiltValue()
abstract class AlbumAudienceSkip implements Built<AlbumAudienceSkip, AlbumAudienceSkipBuilder> {
  /// The requested id that stayed ungranted.
  @BuiltValueField(wireName: r'userId')
  String get userId;

  /// Why the id was skipped: `owner` -- the album's own owner, whose visibility is the ownership row, never a share row; `invalid_account` -- an unknown or disabled account (the same reason for both, so user ids cannot be probed). 
  @BuiltValueField(wireName: r'reason')
  AlbumAudienceSkipReasonEnum get reason;
  // enum reasonEnum {  owner,  invalid_account,  };

  AlbumAudienceSkip._();

  factory AlbumAudienceSkip([void updates(AlbumAudienceSkipBuilder b)]) = _$AlbumAudienceSkip;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(AlbumAudienceSkipBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<AlbumAudienceSkip> get serializer => _$AlbumAudienceSkipSerializer();
}

class _$AlbumAudienceSkipSerializer implements PrimitiveSerializer<AlbumAudienceSkip> {
  @override
  final Iterable<Type> types = const [AlbumAudienceSkip, _$AlbumAudienceSkip];

  @override
  final String wireName = r'AlbumAudienceSkip';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    AlbumAudienceSkip object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'userId';
    yield serializers.serialize(
      object.userId,
      specifiedType: const FullType(String),
    );
    yield r'reason';
    yield serializers.serialize(
      object.reason,
      specifiedType: const FullType(AlbumAudienceSkipReasonEnum),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    AlbumAudienceSkip object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required AlbumAudienceSkipBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'userId':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.userId = valueDes;
          break;
        case r'reason':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(AlbumAudienceSkipReasonEnum),
          ) as AlbumAudienceSkipReasonEnum;
          result.reason = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  AlbumAudienceSkip deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = AlbumAudienceSkipBuilder();
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


/// Why the id was skipped: `owner` -- the album's own owner, whose visibility is the ownership row, never a share row; `invalid_account` -- an unknown or disabled account (the same reason for both, so user ids cannot be probed). 
class AlbumAudienceSkipReasonEnum extends EnumClass {

  @BuiltValueEnumConst(wireName: r'owner')
  static const AlbumAudienceSkipReasonEnum owner = _$albumAudienceSkipReasonEnum_owner;
  @BuiltValueEnumConst(wireName: r'invalid_account')
  static const AlbumAudienceSkipReasonEnum invalidAccount = _$albumAudienceSkipReasonEnum_invalidAccount;

  static Serializer<AlbumAudienceSkipReasonEnum> get serializer => _$albumAudienceSkipReasonEnumSerializer;

  const AlbumAudienceSkipReasonEnum._(String name): super(name);

  static BuiltSet<AlbumAudienceSkipReasonEnum> get values => _$albumAudienceSkipReasonEnumValues;
  static AlbumAudienceSkipReasonEnum valueOf(String name) => _$albumAudienceSkipReasonEnumValueOf(name);
}

