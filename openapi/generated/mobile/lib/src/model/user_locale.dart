//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_collection/built_collection.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'user_locale.g.dart';

/// The account's stored interface language. The values are the UI's installed locales -- the same table the web and mobile clients share -- and the list grows with that table: a locale file added to the i18n directory is a locale this enum admits next. Clients that cannot render a stored value fall back to their own default. 
class UserLocale extends EnumClass {

  @BuiltValueEnumConst(wireName: r'en')
  static const UserLocale en = _$en;
  @BuiltValueEnumConst(wireName: r'zh-Hans')
  static const UserLocale zhHans = _$zhHans;

  static Serializer<UserLocale> get serializer => _$userLocaleSerializer;

  const UserLocale._(String name): super(name);

  static BuiltSet<UserLocale> get values => _$values;
  static UserLocale valueOf(String name) => _$valueOf(name);
}

/// Optionally, enum_class can generate a mixin to go with your enum for use
/// with Angular. It exposes your enum constants as getters. So, if you mix it
/// in to your Dart component class, the values become available to the
/// corresponding Angular template.
///
/// Trigger mixin generation by writing a line like this one next to your enum.
abstract class UserLocaleMixin = Object with _$UserLocaleMixin;

