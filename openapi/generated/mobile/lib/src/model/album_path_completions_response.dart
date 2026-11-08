//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_collection/built_collection.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'album_path_completions_response.g.dart';

/// The path completion answer: the pickable source directories of the creation dialog's combobox. 
///
/// Properties:
/// * [paths] - Full absolute directory paths inside the completion domain -- at or under a configured mount root, outside the server's managed tree, real directories (symlinked entries are never listed). Deduplicated, sorted ascending, at most 20 rows. The empty array is an honest \"nothing matches\", never an error. 
@BuiltValue()
abstract class AlbumPathCompletionsResponse implements Built<AlbumPathCompletionsResponse, AlbumPathCompletionsResponseBuilder> {
  /// Full absolute directory paths inside the completion domain -- at or under a configured mount root, outside the server's managed tree, real directories (symlinked entries are never listed). Deduplicated, sorted ascending, at most 20 rows. The empty array is an honest \"nothing matches\", never an error. 
  @BuiltValueField(wireName: r'paths')
  BuiltList<String> get paths;

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
    yield r'paths';
    yield serializers.serialize(
      object.paths,
      specifiedType: const FullType(BuiltList, [FullType(String)]),
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
        case r'paths':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(BuiltList, [FullType(String)]),
          ) as BuiltList<String>;
          result.paths.replace(valueDes);
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


