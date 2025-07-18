import 'dart:async'; 
import 'dart:convert';
import 'dart:io'; 
import 'dart:math';

import 'package:flutter/foundation.dart';
import 'package:http/http.dart' as http;
import 'package:web_socket_channel/io.dart';
import 'package:web_socket_channel/web_socket_channel.dart';
import '../config/server_config.dart';
import '../models/launch_data.dart';
import '../services/storage_service.dart';

class SyncService {
  final StorageService storageService;
  WebSocketChannel? _channel;
  int _reconnectAttempts = 0;
  late ValueNotifier<bool> _isConnected = ValueNotifier(false);

  ValueNotifier<bool> get isConnected => _isConnected;

  SyncService(this.storageService);

  Future<LaunchData?> fetchSyncData() async {
    final token = await storageService.getToken();
    if (token == null) return null;

    final url = ServerConfig.getSyncUrl();
    print("获取同步数据: $url");
    
    try {
      final response = await http.get(
        Uri.parse(url),
        headers: {'Authorization': token},
      ).timeout(const Duration(seconds: 10));

      if (response.statusCode == 200) {
        final data = LaunchData.fromJson(json.decode(response.body));
        await storageService.saveLaunchData(data);
        return data;
      } else {
        print("获取同步数据失败: ${response.statusCode}");
        return null;
      }
    } on TimeoutException {
      print("获取同步数据超时");
      return null;
    } on SocketException {
      print("无法连接到服务器");
      return null;
    } catch (e) {
      print("获取同步数据异常: $e");
      return null;
    }
  }

  Future<String?> syncData(LaunchData data) async {
    final token = await storageService.getToken();
    if (token == null) return "未认证";

    final url = ServerConfig.getSyncUrl();
    final body = json.encode(data.toJson());
    print("同步数据到: $url");

    try {
      final response = await http.post(
        Uri.parse(url),
        headers: {
          'Authorization': token,
          'Content-Type': 'application/json',
        },
        body: body,
      ).timeout(const Duration(seconds: 10));

      if (response.statusCode == 200) {
        return null; // 成功
      } else {
        return "同步失败: ${response.statusCode}";
      }
    } on TimeoutException {
      return "同步超时";
    } on SocketException {
      return "无法连接到服务器";
    } catch (e) {
      return "网络错误: $e";
    }
  }

  void initWebSocket(Function(LaunchData) onData) async {
    final token = await storageService.getToken();
    if (token == null) return;

    final wsUrl = ServerConfig.getWsUrl();
    print("🛜 尝试连接 WebSocket: $wsUrl");

    try {
      // 关闭现有连接（如果有）
      _channel?.sink.close();
      
      _isConnected.value = true;
_reconnectAttempts = 0;

_isConnected.value = true;
_channel = IOWebSocketChannel.connect(
  Uri.parse(wsUrl).replace(queryParameters: {'token': token}),
  pingInterval: Duration(seconds: 30),
)..sink.done.then((_) => _isConnected.value = false);

// 连接成功后主动同步数据
final latestData = await fetchSyncData();
if (latestData != null) {
  onData(latestData);
}
      _channel!.stream.listen(
        (message) {
          print("收到 WebSocket 消息: $message");
          final data = LaunchData.fromJson(json.decode(message));
          onData(data);
        },
        onError: (error) {
          print("WebSocket错误: $error");
          // 5秒后重试
          final delay = Duration(seconds: min(5 * pow(2, _reconnectAttempts).toInt(), 300));
_reconnectAttempts++;
print('将在${delay.inSeconds}秒后第$_reconnectAttempts次重连');
Future.delayed(delay, () => initWebSocket(onData));
        },
        onDone: () {
          print("WebSocket连接关闭");
          // 5秒后重连
          final delay = Duration(seconds: min(5 * pow(2, _reconnectAttempts).toInt(), 300));
_reconnectAttempts++;
print('将在${delay.inSeconds}秒后第$_reconnectAttempts次重连');
Future.delayed(delay, () => initWebSocket(onData));
        }
      );
    } catch (e) {
      print("WebSocket连接异常: $e");
      // 5秒后重试
      final delay = Duration(seconds: min(5 * pow(2, _reconnectAttempts).toInt(), 300));
_reconnectAttempts++;
print('将在${delay.inSeconds}秒后第$_reconnectAttempts次重连');
Future.delayed(delay, () => initWebSocket(onData));
    }
  }

  void closeWebSocket() {
    _channel?.sink.close();
  }
}