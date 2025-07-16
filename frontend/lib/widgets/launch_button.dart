import 'package:flutter/material.dart';
import '../services/theme_service.dart';

class LaunchButton extends StatelessWidget {
  final VoidCallback onPressed;

  const LaunchButton({Key? key, required this.onPressed}) : super(key: key);

  @override
  Widget build(BuildContext context) {
    return FloatingActionButton(
      backgroundColor: ThemeService.getButtonColor(context),
      onPressed: onPressed,
      child: const Icon(Icons.rocket_launch, size: 36),
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(50)),
    );
  }
}