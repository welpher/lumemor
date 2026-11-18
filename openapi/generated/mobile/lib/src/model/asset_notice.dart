//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_collection/built_collection.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'asset_notice.g.dart';

/// One thing the client MUST tell the user about a successful upload: the server made a decision the user cannot see in the response's asset fields. Notices are informational, never failures -- the upload itself succeeded. 
///
/// Properties:
/// * [code] - What happened.  `LIVE_PHOTO_MOTION_ABSORBED`: the Live Photo's motion bytes already existed as a live standalone video of this owner; that row became the pair's motion (`assetId` names it) and no longer lists as its own asset. Carried on a 201 `created` answer.  `LIVE_PHOTO_MOTION_ALREADY_STORED`: the uploaded bytes already play as a Live Photo's motion, so the returned row is that hidden component (`assetId` names it) and does not list on its own. Carried on a 200 `duplicate` answer: the content genuinely exists, so the state stays `duplicate` -- there is nothing for the user to fix. 
/// * [assetId] - The asset the notice is about: the video row that plays as a Live Photo's motion. 
/// * [name] - The affected asset's original filename, so the message can name what moved. 
@BuiltValue()
abstract class AssetNotice implements Built<AssetNotice, AssetNoticeBuilder> {
  /// What happened.  `LIVE_PHOTO_MOTION_ABSORBED`: the Live Photo's motion bytes already existed as a live standalone video of this owner; that row became the pair's motion (`assetId` names it) and no longer lists as its own asset. Carried on a 201 `created` answer.  `LIVE_PHOTO_MOTION_ALREADY_STORED`: the uploaded bytes already play as a Live Photo's motion, so the returned row is that hidden component (`assetId` names it) and does not list on its own. Carried on a 200 `duplicate` answer: the content genuinely exists, so the state stays `duplicate` -- there is nothing for the user to fix. 
  @BuiltValueField(wireName: r'code')
  AssetNoticeCodeEnum get code;
  // enum codeEnum {  LIVE_PHOTO_MOTION_ABSORBED,  LIVE_PHOTO_MOTION_ALREADY_STORED,  };

  /// The asset the notice is about: the video row that plays as a Live Photo's motion. 
  @BuiltValueField(wireName: r'assetId')
  String get assetId;

  /// The affected asset's original filename, so the message can name what moved. 
  @BuiltValueField(wireName: r'name')
  String get name;

  AssetNotice._();

  factory AssetNotice([void updates(AssetNoticeBuilder b)]) = _$AssetNotice;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(AssetNoticeBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<AssetNotice> get serializer => _$AssetNoticeSerializer();
}

class _$AssetNoticeSerializer implements PrimitiveSerializer<AssetNotice> {
  @override
  final Iterable<Type> types = const [AssetNotice, _$AssetNotice];

  @override
  final String wireName = r'AssetNotice';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    AssetNotice object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'code';
    yield serializers.serialize(
      object.code,
      specifiedType: const FullType(AssetNoticeCodeEnum),
    );
    yield r'assetId';
    yield serializers.serialize(
      object.assetId,
      specifiedType: const FullType(String),
    );
    yield r'name';
    yield serializers.serialize(
      object.name,
      specifiedType: const FullType(String),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    AssetNotice object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required AssetNoticeBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'code':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(AssetNoticeCodeEnum),
          ) as AssetNoticeCodeEnum;
          result.code = valueDes;
          break;
        case r'assetId':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.assetId = valueDes;
          break;
        case r'name':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.name = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  AssetNotice deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = AssetNoticeBuilder();
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


/// What happened.  `LIVE_PHOTO_MOTION_ABSORBED`: the Live Photo's motion bytes already existed as a live standalone video of this owner; that row became the pair's motion (`assetId` names it) and no longer lists as its own asset. Carried on a 201 `created` answer.  `LIVE_PHOTO_MOTION_ALREADY_STORED`: the uploaded bytes already play as a Live Photo's motion, so the returned row is that hidden component (`assetId` names it) and does not list on its own. Carried on a 200 `duplicate` answer: the content genuinely exists, so the state stays `duplicate` -- there is nothing for the user to fix. 
class AssetNoticeCodeEnum extends EnumClass {

  @BuiltValueEnumConst(wireName: r'LIVE_PHOTO_MOTION_ABSORBED')
  static const AssetNoticeCodeEnum AssetNoticeCodeLivePhotoMotionAbsorbed = _$assetNoticeCodeEnum_AssetNoticeCodeLivePhotoMotionAbsorbed;
  @BuiltValueEnumConst(wireName: r'LIVE_PHOTO_MOTION_ALREADY_STORED')
  static const AssetNoticeCodeEnum AssetNoticeCodeLivePhotoMotionAlreadyStored = _$assetNoticeCodeEnum_AssetNoticeCodeLivePhotoMotionAlreadyStored;

  static Serializer<AssetNoticeCodeEnum> get serializer => _$assetNoticeCodeEnumSerializer;

  const AssetNoticeCodeEnum._(String name): super(name);

  static BuiltSet<AssetNoticeCodeEnum> get values => _$assetNoticeCodeEnumValues;
  static AssetNoticeCodeEnum valueOf(String name) => _$assetNoticeCodeEnumValueOf(name);
}

