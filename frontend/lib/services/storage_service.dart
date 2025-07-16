import 'dart:convert';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:shared_preferences/shared_preferences.dart';
import '../models/launch_data.dart';

class StorageService {
  static const _tokenKey = 'auth_token';
  static const _launchDataKey = 'launch_data';
  static const _usernameKey = 'username';
  static const _userIdKey = 'user_id'; // 新增用户ID键

  final FlutterSecureStorage _secureStorage = const FlutterSecureStorage();
  final SharedPreferences _prefs;

  StorageService(this._prefs);

  Future<void> saveUserId(int userId) async {
    await _prefs.setInt(_userIdKey, userId);
  }

  int? getUserId() {
    return _prefs.getInt(_userIdKey);
  }

  Future<void> saveToken(String token) async {
  // 确保令牌正确存储
  token = token.trim();
  await _secureStorage.write(key: _tokenKey, value: token);
  }

  Future<String?> getToken() async {
    return await _secureStorage.read(key: _tokenKey);
  }

  Future<void> clearToken() async {
    await _secureStorage.delete(key: _tokenKey);
  }

  Future<void> saveUsername(String username) async {
    await _prefs.setString(_usernameKey, username);
  }

  String? getUsername() {
    return _prefs.getString(_usernameKey);
  }

  Future<void> saveLaunchData(LaunchData data) async {
    final jsonData = json.encode(data.toJson());
    await _prefs.setString(_launchDataKey, jsonData);
  }

  LaunchData? getLaunchData() {
    final jsonData = _prefs.getString(_launchDataKey);
    if (jsonData == null) return null;
    return LaunchData.fromJson(json.decode(jsonData));
  }
}