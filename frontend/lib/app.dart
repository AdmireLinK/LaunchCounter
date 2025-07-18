import 'package:flutter/material.dart';
import 'pages/auth_page.dart';
import 'pages/main_page.dart';
import 'services/storage_service.dart';

class App extends StatefulWidget {
  final StorageService storageService;

  const App({Key? key, required this.storageService}) : super(key: key);

  @override
  _AppState createState() => _AppState();
}

class _AppState extends State<App> {
  bool _isAuthenticated = false;
  bool _isLoading = true;

  @override
  void initState() {
    super.initState();
    _checkAuth();
  }

  Future<void> _checkAuth() async {
    final token = await widget.storageService.getToken();
    setState(() {
      _isAuthenticated = token != null;
      _isLoading = false;
    });
  }

  @override
  Widget build(BuildContext context) {
    if (_isLoading) {
      return Scaffold(body: Center(child: CircularProgressIndicator()));
    }

    return _isAuthenticated
        ? MainPage(storageService: widget.storageService)
        : AuthPage(storageService: widget.storageService);
  }
}