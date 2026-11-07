//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:lumemor_api/src/model/album_summary.dart';
import 'package:built_collection/built_collection.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'album_wall_response.g.dart';

/// The caller's album wall: the own group followed by the shared group, concatenated into one array. 
///
/// Properties:
/// * [albums] - Every wall album: the pinned sync card first (the caller's own core upload stream, the default \"My photos\" album), then the own group (the caller's virtual and directory albums, `updatedAt` descending), then the shared group (the albums shared to the caller, the caller's grant time descending). Each group breaks ties on the album id, descending. A sync album another account owns never appears. 
@BuiltValue()
abstract class AlbumWallResponse implements Built<AlbumWallResponse, AlbumWallResponseBuilder> {
  /// Every wall album: the pinned sync card first (the caller's own core upload stream, the default \"My photos\" album), then the own group (the caller's virtual and directory albums, `updatedAt` descending), then the shared group (the albums shared to the caller, the caller's grant time descending). Each group breaks ties on the album id, descending. A sync album another account owns never appears. 
  @BuiltValueField(wireName: r'albums')
  BuiltList<AlbumSummary> get albums;

  AlbumWallResponse._();

  factory AlbumWallResponse([void updates(AlbumWallResponseBuilder b)]) = _$AlbumWallResponse;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(AlbumWallResponseBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<AlbumWallResponse> get serializer => _$AlbumWallResponseSerializer();
}

class _$AlbumWallResponseSerializer implements PrimitiveSerializer<AlbumWallResponse> {
  @override
  final Iterable<Type> types = const [AlbumWallResponse, _$AlbumWallResponse];

  @override
  final String wireName = r'AlbumWallResponse';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    AlbumWallResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'albums';
    yield serializers.serialize(
      object.albums,
      specifiedType: const FullType(BuiltList, [FullType(AlbumSummary)]),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    AlbumWallResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required AlbumWallResponseBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'albums':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(BuiltList, [FullType(AlbumSummary)]),
          ) as BuiltList<AlbumSummary>;
          result.albums.replace(valueDes);
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  AlbumWallResponse deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = AlbumWallResponseBuilder();
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


