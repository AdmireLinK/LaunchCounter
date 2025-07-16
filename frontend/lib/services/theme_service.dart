import 'package:flutter/material.dart';

class ThemeService {
  static bool isDarkMode(BuildContext context) {
    return MediaQuery.of(context).platformBrightness == Brightness.dark;
  }

  static Color getBackgroundColor(BuildContext context) {
    return isDarkMode(context) ? Colors.grey[900]! : Colors.white;
  }

  static Color getTextColor(BuildContext context) {
    return isDarkMode(context) ? Colors.white : Colors.black;
  }

  static Color getButtonColor(BuildContext context) {
    return isDarkMode(context) ? Colors.blue[800]! : const Color(0xFF66CCFF);
  }
}