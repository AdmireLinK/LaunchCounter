class LaunchData {
  final int userId; // 使用 final 确保不可变
  int total;
  Map<String, int> yearData;
  Map<String, int> monthData;
  Map<String, int> dayData;
  DateTime lastLaunch;

  LaunchData({
    required this.userId,
    required this.total,
    required this.yearData,
    required this.monthData,
    required this.dayData,
    required this.lastLaunch,
  });

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

  // 添加 copyWith 方法
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

  void increment() {
    final now = DateTime.now();
    total++;
    _updateCounter(yearData, now.year.toString());
    _updateCounter(monthData, "${now.year}-${now.month}");
    _updateCounter(dayData, "${now.year}-${now.month}-${now.day}");
    lastLaunch = now;
  }

  void _updateCounter(Map<String, int> map, String key) {
    map[key] = (map[key] ?? 0) + 1;
  }
}