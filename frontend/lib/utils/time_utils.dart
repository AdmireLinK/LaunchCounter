class TimeUtils {
  static String formatDuration(Duration duration) {
    if (duration.inDays > 365) return '超过1年';
    if (duration.inDays > 0) return '${duration.inDays}天';
    if (duration.inHours > 0) return '${duration.inHours}小时';
    if (duration.inMinutes > 0) return '${duration.inMinutes}分钟';
    return '${duration.inSeconds}秒';
  }

  static String getTodayKey() {
    final now = DateTime.now();
    return "${now.year}-${now.month}-${now.day}";
  }

  static String getThisMonthKey() {
    final now = DateTime.now();
    return "${now.year}-${now.month}";
  }

  static String getThisYearKey() {
    return DateTime.now().year.toString();
  }
}