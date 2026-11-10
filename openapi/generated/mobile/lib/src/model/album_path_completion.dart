//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'album_path_completion.g.dart';

/// One row of the completion answer: the directory's path and the write fact the listing's probe discovered. The probe is the creation-time probe's own three steps (write, read back, remove) run once per row, so the dialog knows -- before any album is created -- whether the deletability switch applies there. 
///
/// Properties:
/// * [path] - Full absolute directory path of the container's visible filesystem, never inside the server's managed tree. 
/// * [writable] - The directory's write fact as the listing discovered it: true when the probe file landed, read back and removed cleanly; false when any step refused. The fact is information, never an error -- a false row lists alongside the true ones, and picking it creates a legitimate read-only-source album. 
@BuiltValue()
abstract class AlbumPathCompletion implements Built<AlbumPathCompletion, AlbumPathCompletionBuilder> {
  /// Full absolute directory path of the container's visible filesystem, never inside the server's managed tree. 
  @BuiltValueField(wireName: r'path')
  String get path;

  /// The directory's write fact as the listing discovered it: true when the probe file landed, read back and removed cleanly; false when any step refused. The fact is information, never an error -- a false row lists alongside the true ones, and picking it creates a legitimate read-only-source album. 
  @BuiltValueField(wireName: r'writable')
  bool get writable;

  AlbumPathCompletion._();

  factory AlbumPathCompletion([void updates(AlbumPathCompletionBuilder b)]) = _$AlbumPathCompletion;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(AlbumPathCompletionBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<AlbumPathCompletion> get serializer => _$AlbumPathCompletionSerializer();
}

class _$AlbumPathCompletionSerializer implements PrimitiveSerializer<AlbumPathCompletion> {
  @override
  final Iterable<Type> types = const [AlbumPathCompletion, _$AlbumPathCompletion];

  @override
  final String wireName = r'AlbumPathCompletion';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    AlbumPathCompletion object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'path';
    yield serializers.serialize(
      object.path,
      specifiedType: const FullType(String),
    );
    yield r'writable';
    yield serializers.serialize(
      object.writable,
      specifiedType: const FullType(bool),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    AlbumPathCompletion object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required AlbumPathCompletionBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'path':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.path = valueDes;
          break;
        case r'writable':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(bool),
          ) as bool;
          result.writable = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  AlbumPathCompletion deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = AlbumPathCompletionBuilder();
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


