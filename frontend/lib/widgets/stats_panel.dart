import 'package:flutter/material.dart';
import '../models/launch_data.dart';
import '../services/theme_service.dart';
import '../utils/time_utils.dart';

class StatsPanel extends StatelessWidget {
  final LaunchData data;

  const StatsPanel({Key? key, required this.data}) : super(key: key);

  @override
  Widget build(BuildContext context) {
    final now = DateTime.now();
    final lastLaunchDiff = data.lastLaunch != DateTime(0)
        ? now.difference(data.lastLaunch)
        : Duration(days: 999);

    return Card(
      elevation: 4,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(16)),
      child: Padding(
        padding: const EdgeInsets.all(20.0),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              '发射状态',
              style: TextStyle(
                fontSize: 20,
                fontWeight: FontWeight.bold,
                color: ThemeService.getTextColor(context),
              ),
            ),
            const SizedBox(height: 15),
            _buildStatItem(context, '距离上次发射', TimeUtils.formatDuration(lastLaunchDiff)),
            const Divider(),
            _buildStatItem(context, '总发射次数', '${data.total}'),
            const Divider(),
            _buildStatItem(context, '本年发射', '${data.yearData[TimeUtils.getThisYearKey()] ?? 0}'),
            const Divider(),
            _buildStatItem(context, '本月发射', '${data.monthData[TimeUtils.getThisMonthKey()] ?? 0}'),
            const Divider(),
            _buildStatItem(context, '今日发射', '${data.dayData[TimeUtils.getTodayKey()] ?? 0}'),
          ],
        ),
      ),
    );
  }

  Widget _buildStatItem(BuildContext context, String label, String value) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 8.0),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.spaceBetween,
        children: [
          Text(
            label,
            style: TextStyle(
              fontSize: 16,
              color: ThemeService.getTextColor(context),
            ),
          ),
          Text(
            value,
            style: TextStyle(
              fontSize: 18,
              fontWeight: FontWeight.bold,
              color: ThemeService.getButtonColor(context),
            ),
          ),
        ],
      ),
    );
  }
}