import 'dart:async'; // 添加 Timer 所需的包
import 'package:flutter/material.dart';
import '../services/sync_service.dart';
import '../services/storage_service.dart';
import '../models/launch_data.dart';
import '../widgets/stats_panel.dart';
import '../widgets/launch_button.dart'; // 导入 LaunchButton
import '../widgets/sync_indicator.dart';
import '../services/theme_service.dart';
import '../utils/time_utils.dart';
import '../app.dart'; // 导入 App

class MainPage extends StatefulWidget {
  final StorageService storageService;

  const MainPage({Key? key, required this.storageService}) : super(key: key);

  @override
  _MainPageState createState() => _MainPageState();
}

class _MainPageState extends State<MainPage> {
  late LaunchData _launchData;

  late SyncService _syncService;
  String? _syncStatus;
  bool _syncError = false;
  Timer? _timer;

  @override
  void initState() {
    super.initState();

    // 获取存储的用户ID
    final userId = widget.storageService.getUserId() ?? 0;

    // 获取存储的发射数据或创建新实例
    final storedData = widget.storageService.getLaunchData();
    
    // 初始化时设置 userId
    _launchData = storedData != null
        ? storedData.copyWith(userId: userId) // 使用 copyWith 方法更新用户ID
        : LaunchData.empty().copyWith(userId: userId);
    
    _launchData = widget.storageService.getLaunchData() ?? LaunchData.empty();
    _syncService = SyncService(widget.storageService);
    _initData();
    _startTimer();
  }

  void _startTimer() {
    _timer = Timer.periodic(Duration(minutes: 1), (timer) {
      if (mounted) {
        setState(() {});
      }
    });
  }

  Future<void> _initData() async {
    // 从服务器获取最新数据
    final remoteData = await _syncService.fetchSyncData();
    if (remoteData != null) {
      setState(() {
        _launchData = remoteData;
      });
      await widget.storageService.saveLaunchData(_launchData);
    }

    // 初始化WebSocket
    _syncService.initWebSocket((data) {
      setState(() {
        _launchData = data;
      });
      widget.storageService.saveLaunchData(data);
      _showSyncStatus('数据已同步');
    });
  }

  @override
  void dispose() {
    _syncService.closeWebSocket();
    _timer?.cancel();
    super.dispose();
  }

  void _showSyncStatus(String message, {bool isError = false}) {
    setState(() {
      _syncStatus = message;
      _syncError = isError;
    });
    Future.delayed(Duration(seconds: 3), () {
      if (mounted) {
        setState(() {
          _syncStatus = null;
        });
      }
    });
  }

  Future<void> _onLaunch() async {
    setState(() {
      _launchData.increment();
    });
    await widget.storageService.saveLaunchData(_launchData);

    // 尝试同步
    final error = await _syncService.syncData(_launchData);
    if (error != null) {
      _showSyncStatus(error, isError: true);
    } else {
      _showSyncStatus('同步成功');
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: ThemeService.getBackgroundColor(context),
      appBar: AppBar(
        title: Text('发射计数器', style: TextStyle(color: ThemeService.getTextColor(context))),
        backgroundColor: Colors.transparent,
        elevation: 0,
        actions: [
          Padding(
            padding: EdgeInsets.only(right: 20),
            child: ValueListenableBuilder<bool>(
              valueListenable: _syncService.isConnected,
              builder: (_, isConnected, __) => Text(
                '状态: ${isConnected ? '已连接' : '断开'}',
                style: TextStyle(color: isConnected ? Colors.green : Colors.red)
              ),
            ),
          ),
          IconButton(
            icon: Icon(Icons.logout),
            onPressed: () async {
              await widget.storageService.clearToken();
              Navigator.of(context).pushReplacement(
                MaterialPageRoute(builder: (_) => App(storageService: widget.storageService)),
              );
            },
          ),
        ],
      ),
      body: Stack(
        children: [
          SingleChildScrollView(
            child: Padding(
              padding: const EdgeInsets.all(16.0),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  StatsPanel(data: _launchData),
                  const SizedBox(height: 30),
                  Text(
                    '发射统计',
                    style: TextStyle(
                      fontSize: 20,
                      fontWeight: FontWeight.bold,
                      color: ThemeService.getTextColor(context),
                    ),
                  ),
                  const SizedBox(height: 10),
                  _buildStatCard('今日', _launchData.dayData[TimeUtils.getTodayKey()] ?? 0),
                  const SizedBox(height: 10),
                  _buildStatCard('本月', _launchData.monthData[TimeUtils.getThisMonthKey()] ?? 0),
                  const SizedBox(height: 10),
                  _buildStatCard('今年', _launchData.yearData[TimeUtils.getThisYearKey()] ?? 0),
                  const SizedBox(height: 10),
                  _buildStatCard('总计', _launchData.total),
                  const SizedBox(height: 80),
                ],
              ),
            ),
          ),
          if (_syncStatus != null)
            Positioned(
              top: 70,
              left: 0,
              right: 0,
              child: Center(
                child: SyncIndicator(
                  status: _syncStatus!,
                  isError: _syncError,
                ),
              ),
            ),
        ],
      ),
      floatingActionButton: LaunchButton(onPressed: _onLaunch),
      floatingActionButtonLocation: FloatingActionButtonLocation.centerFloat,
    );
  }

  Widget _buildStatCard(String title, int count) {
    return Card(
      elevation: 4,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
      child: Padding(
        padding: const EdgeInsets.all(16.0),
        child: Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            Text(
              title,
              style: TextStyle(
                fontSize: 18,
                fontWeight: FontWeight.bold,
                color: ThemeService.getTextColor(context),
              ),
            ),
            Text(
              '$count',
              style: TextStyle(
                fontSize: 24,
                fontWeight: FontWeight.bold,
                color: ThemeService.getButtonColor(context),
              ),
            ),
          ],
        ),
      ),
    );
  }
}