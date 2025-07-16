import 'dart:async'; // 添加超时异常
import 'dart:convert';
import 'dart:io'; // 添加 SocketException

import 'package:http/http.dart' as http;
import '../config/server_config.dart';
import '../services/storage_service.dart';

class AuthService {
  final StorageService storageService;

  AuthService(this.storageService);

  Future<String?> registerOrLogin(String username, String password, bool isLogin) async {
    final url = isLogin ? ServerConfig.getLoginUrl() : ServerConfig.getRegisterUrl();
    print("请求URL: $url");

    try {
      // 添加超时设置
      final response = await http.post(
        Uri.parse(url),
        headers: {'Content-Type': 'application/json'},
        body: json.encode({'username': username, 'password': password}),
      ).timeout(const Duration(seconds: 10)); // 10秒超时

      if (response.statusCode == 200) {
        final token = json.decode(response.body)['token'];
        await storageService.saveToken(token);
        await storageService.saveUsername(username);
        return null; // 成功
      } else {
        final error = json.decode(response.body)['error'] ?? '未知错误';
        return error;
      }
    } on TimeoutException catch (e) {
      print("请求超时: $e");
      return '连接超时，请稍后重试';
    } on SocketException catch (e) {
      print("网络连接错误: $e");
      return '无法连接到服务器，请检查网络';
    } on http.ClientException catch (e) {
      print("客户端异常: $e");
      return '网络错误: ${e.message}';
    } catch (e) {
      print("未知错误: $e");
      return '网络连接失败';
    }
  }
  
  // 删除未使用的 _getUserIdFromToken 方法
}