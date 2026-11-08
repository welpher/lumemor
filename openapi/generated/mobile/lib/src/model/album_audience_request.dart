//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_collection/built_collection.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'album_audience_request.g.dart';

/// The audience write body: the COMPLETE grantee set, the state the album's `album_shares` rows must converge to. The field must ride the body -- an explicit empty array IS the valid \"only me\" request (every row leaves), while an absent field is a client bug the 400 names, the destructive-silent- default guard of the settings writes. 
///
/// Properties:
/// * [audience] - Every account that may view the album after this write, between 0 and 200 entries. Repeated ids collapse to one application. The owner's own id is skippable noise (the owner sees the album by construction), and ids that name no usable account are skipped with a reason -- never errors. 
@BuiltValue()
abstract class AlbumAudienceRequest implements Built<AlbumAudienceRequest, AlbumAudienceRequestBuilder> {
  /// Every account that may view the album after this write, between 0 and 200 entries. Repeated ids collapse to one application. The owner's own id is skippable noise (the owner sees the album by construction), and ids that name no usable account are skipped with a reason -- never errors. 
  @BuiltValueField(wireName: r'audience')
  BuiltList<String> get audience;

  AlbumAudienceRequest._();

  factory AlbumAudienceRequest([void updates(AlbumAudienceRequestBuilder b)]) = _$AlbumAudienceRequest;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(AlbumAudienceRequestBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<AlbumAudienceRequest> get serializer => _$AlbumAudienceRequestSerializer();
}

class _$AlbumAudienceRequestSerializer implements PrimitiveSerializer<AlbumAudienceRequest> {
  @override
  final Iterable<Type> types = const [AlbumAudienceRequest, _$AlbumAudienceRequest];

  @override
  final String wireName = r'AlbumAudienceRequest';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    AlbumAudienceRequest object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'audience';
    yield serializers.serialize(
      object.audience,
      specifiedType: const FullType(BuiltList, [FullType(String)]),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    AlbumAudienceRequest object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required AlbumAudienceRequestBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'audience':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(BuiltList, [FullType(String)]),
          ) as BuiltList<String>;
          result.audience.replace(valueDes);
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  AlbumAudienceRequest deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = AlbumAudienceRequestBuilder();
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


