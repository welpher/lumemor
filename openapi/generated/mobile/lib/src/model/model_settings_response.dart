//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_collection/built_collection.dart';
import 'package:lumemor_api/src/model/model_params.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'model_settings_response.g.dart';

/// The model store's policy, download sources and pipeline params of `GET /admin/settings/models` and the successful `PUT`. 
///
/// Properties:
/// * [autoDownload] - Whether an AI job whose slot is configured but whose weights are missing may start the model group's download on its own. It never fills an unconfigured slot: the first-run download stays the administrator's own act. 
/// * [allowedHosts] - The hosts the model download may travel through. A URL whose host matches no entry -- entries are exact hostnames or a `*.` wildcard prefix for a whole domain suffix -- fails the download with the host named, and the answer points at the sources panel that can add it. The shipped default list covers the manifest's own hosts; the manifest's pinned URLs stay the identity of what is downloaded, this list is only where the bytes may come from. 
/// * [mirrorBaseUrl] - The download mirror's base host, the empty string when no mirror is configured. Non-empty, it replaces the manifest's pinned `huggingface.co` hostname wholesale on every candidate URL before the fetch, and its host joins `allowedHosts` automatically at save time (a mirror the list refuses would fail on its first hop). Custom model imports are not mirror-rewritten. 
/// * [params] 
@BuiltValue()
abstract class ModelSettingsResponse implements Built<ModelSettingsResponse, ModelSettingsResponseBuilder> {
  /// Whether an AI job whose slot is configured but whose weights are missing may start the model group's download on its own. It never fills an unconfigured slot: the first-run download stays the administrator's own act. 
  @BuiltValueField(wireName: r'autoDownload')
  bool get autoDownload;

  /// The hosts the model download may travel through. A URL whose host matches no entry -- entries are exact hostnames or a `*.` wildcard prefix for a whole domain suffix -- fails the download with the host named, and the answer points at the sources panel that can add it. The shipped default list covers the manifest's own hosts; the manifest's pinned URLs stay the identity of what is downloaded, this list is only where the bytes may come from. 
  @BuiltValueField(wireName: r'allowedHosts')
  BuiltList<String> get allowedHosts;

  /// The download mirror's base host, the empty string when no mirror is configured. Non-empty, it replaces the manifest's pinned `huggingface.co` hostname wholesale on every candidate URL before the fetch, and its host joins `allowedHosts` automatically at save time (a mirror the list refuses would fail on its first hop). Custom model imports are not mirror-rewritten. 
  @BuiltValueField(wireName: r'mirrorBaseUrl')
  String get mirrorBaseUrl;

  @BuiltValueField(wireName: r'params')
  ModelParams get params;

  ModelSettingsResponse._();

  factory ModelSettingsResponse([void updates(ModelSettingsResponseBuilder b)]) = _$ModelSettingsResponse;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(ModelSettingsResponseBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<ModelSettingsResponse> get serializer => _$ModelSettingsResponseSerializer();
}

class _$ModelSettingsResponseSerializer implements PrimitiveSerializer<ModelSettingsResponse> {
  @override
  final Iterable<Type> types = const [ModelSettingsResponse, _$ModelSettingsResponse];

  @override
  final String wireName = r'ModelSettingsResponse';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    ModelSettingsResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'autoDownload';
    yield serializers.serialize(
      object.autoDownload,
      specifiedType: const FullType(bool),
    );
    yield r'allowedHosts';
    yield serializers.serialize(
      object.allowedHosts,
      specifiedType: const FullType(BuiltList, [FullType(String)]),
    );
    yield r'mirrorBaseUrl';
    yield serializers.serialize(
      object.mirrorBaseUrl,
      specifiedType: const FullType(String),
    );
    yield r'params';
    yield serializers.serialize(
      object.params,
      specifiedType: const FullType(ModelParams),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    ModelSettingsResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required ModelSettingsResponseBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'autoDownload':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(bool),
          ) as bool;
          result.autoDownload = valueDes;
          break;
        case r'allowedHosts':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(BuiltList, [FullType(String)]),
          ) as BuiltList<String>;
          result.allowedHosts.replace(valueDes);
          break;
        case r'mirrorBaseUrl':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.mirrorBaseUrl = valueDes;
          break;
        case r'params':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(ModelParams),
          ) as ModelParams;
          result.params.replace(valueDes);
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  ModelSettingsResponse deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = ModelSettingsResponseBuilder();
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


