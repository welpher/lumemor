//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:lumemor_api/src/model/album_path_completion.dart';
import 'package:built_collection/built_collection.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'album_path_completions_response.g.dart';

/// The path completion answer: the pickable source directories of the creation dialog's combobox, each with the write fact the completion discovered by probing the directory once (the write-probe discipline of the creation, applied at listing time -- the dialog renders the deletability switch only for a `writable` row). 
///
/// Properties:
/// * [completions] - The pickable directories of the container's visible filesystem -- outside the server's managed tree, real directories (symlinked entries are never listed). Deduplicated, sorted ascending by path, at most 20 rows. The empty array is an honest \"nothing matches\", never an error. Every row carries its write fact: the probe file was written, read back and removed cleanly (`writable` true), or any step refused (false -- a read-only directory is a legitimate source, and a false fact is information, never an error). 
@BuiltValue()
abstract class AlbumPathCompletionsResponse implements Built<AlbumPathCompletionsResponse, AlbumPathCompletionsResponseBuilder> {
  /// The pickable directories of the container's visible filesystem -- outside the server's managed tree, real directories (symlinked entries are never listed). Deduplicated, sorted ascending by path, at most 20 rows. The empty array is an honest \"nothing matches\", never an error. Every row carries its write fact: the probe file was written, read back and removed cleanly (`writable` true), or any step refused (false -- a read-only directory is a legitimate source, and a false fact is information, never an error). 
  @BuiltValueField(wireName: r'completions')
  BuiltList<AlbumPathCompletion> get completions;

  AlbumPathCompletionsResponse._();

  factory AlbumPathCompletionsResponse([void updates(AlbumPathCompletionsResponseBuilder b)]) = _$AlbumPathCompletionsResponse;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(AlbumPathCompletionsResponseBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<AlbumPathCompletionsResponse> get serializer => _$AlbumPathCompletionsResponseSerializer();
}

class _$AlbumPathCompletionsResponseSerializer implements PrimitiveSerializer<AlbumPathCompletionsResponse> {
  @override
  final Iterable<Type> types = const [AlbumPathCompletionsResponse, _$AlbumPathCompletionsResponse];

  @override
  final String wireName = r'AlbumPathCompletionsResponse';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    AlbumPathCompletionsResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'completions';
    yield serializers.serialize(
      object.completions,
      specifiedType: const FullType(BuiltList, [FullType(AlbumPathCompletion)]),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    AlbumPathCompletionsResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required AlbumPathCompletionsResponseBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'completions':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(BuiltList, [FullType(AlbumPathCompletion)]),
          ) as BuiltList<AlbumPathCompletion>;
          result.completions.replace(valueDes);
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  AlbumPathCompletionsResponse deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = AlbumPathCompletionsResponseBuilder();
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


