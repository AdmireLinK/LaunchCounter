class LaunchData {
  /// 用户唯一标识（不可变）
  final int userId;
  /// 总发射次数统计
  int total;
  /// 按年份统计的发射记录（格式：年份字符串 -> 次数）
  Map<String, int> yearData;
  /// 按月统计的发射记录（格式：年-月 -> 次数）
  Map<String, int> monthData;
  /// 按日统计的发射记录（格式：年-月-日 -> 次数）
  Map<String, int> dayData;
  /// 最后一次发射时间
  DateTime lastLaunch;

  /// 主构造函数 - 所有参数均为必需参数
  LaunchData({
    required this.userId,
    required this.total,
    required this.yearData,
    required this.monthData,
    required this.dayData,
    required this.lastLaunch,
  });

  /// 创建空数据对象的工厂方法
  factory LaunchData.empty() {
    return LaunchData(
      userId: 0,
      total: 0,
      yearData: {},
      monthData: {},
      dayData: {},
      lastLaunch: DateTime(0),
    );
  }

  /// 从 JSON 数据解析的工厂方法
  factory LaunchData.fromJson(Map<String, dynamic> json) {
    return LaunchData(
      userId: json['user_id'] is int ? json['user_id'] : int.tryParse(json['user_id']?.toString() ?? '0') ?? 0,
      total: json['total'] ?? 0,
      yearData: Map<String, int>.from(json['year_data'] ?? {}),
      monthData: Map<String, int>.from(json['month_data'] ?? {}),
      dayData: Map<String, int>.from(json['day_data'] ?? {}),
      lastLaunch: json['last_launch'] != null
          ? DateTime.parse(json['last_launch'])
          : DateTime(0),
    );
  }

  /// 转换为 JSON 格式（用于持久化存储）
  Map<String, dynamic> toJson() {
    return {
    'user_id': userId,
    'total': total,
    'year_data': yearData,
    'month_data': monthData,
    'day_data': dayData,
    'last_launch': lastLaunch.toUtc().toIso8601String(), // 使用 UTC 时间
  };
}

  /// 对象复制方法（支持部分属性覆盖）
  LaunchData copyWith({
    int? userId,
    int? total,
    Map<String, int>? yearData,
    Map<String, int>? monthData,
    Map<String, int>? dayData,
    DateTime? lastLaunch,
  }) {
    return LaunchData(
      userId: userId ?? this.userId,
      total: total ?? this.total,
      yearData: yearData ?? Map.from(this.yearData),
      monthData: monthData ?? Map.from(this.monthData),
      dayData: dayData ?? Map.from(this.dayData),
      lastLaunch: lastLaunch ?? this.lastLaunch,
    );
  }

  /// 增加发射次数（更新所有统计维度）
  void increment() {
    final now = DateTime.now();
    total++;
    _updateCounter(yearData, now.year.toString());
    _updateCounter(monthData, "${now.year}-${now.month}");
    _updateCounter(dayData, "${now.year}-${now.month}-${now.day}");
    lastLaunch = now;
  }

  /// 私有方法：更新指定统计维度的计数器
  void _updateCounter(Map<String, int> map, String key) {
    map[key] = (map[key] ?? 0) + 1;
  }
}