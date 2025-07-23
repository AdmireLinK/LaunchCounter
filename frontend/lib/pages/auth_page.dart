import 'package:flutter/material.dart';
import '../services/auth_service.dart';
import '../services/storage_service.dart';
import '../app.dart';
import '../services/theme_service.dart';

// 认证页面组件，处理用户登录逻辑
class AuthPage extends StatefulWidget {
  /// 本地存储服务实例，用于持久化认证信息
  final StorageService storageService;

  /// 构造函数接收必需的存储服务实例
  const AuthPage({Key? key, required this.storageService}) : super(key: key);

  @override
  _AuthPageState createState() => _AuthPageState();
}

// 认证页面状态管理类
class _AuthPageState extends State<AuthPage> {
  /// 表单验证键，用于控制表单状态
  final _formKey = GlobalKey<FormState>();
  /// 用户名输入控制器，管理文本输入内容
  final _usernameController = TextEditingController();
  /// 密码输入控制器，管理敏感信息输入
  final _passwordController = TextEditingController();
  
  /// 加载状态标识，控制进度指示器显示
  bool _isLoading = false;
  /// 错误信息提示，展示认证失败原因
  String? _errorMessage;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: ThemeService.getBackgroundColor(context),
      body: Padding(
        padding: const EdgeInsets.all(16.0),
        child: Form(
          key: _formKey,
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              // 品牌图标展示
              Icon(Icons.rocket_launch, size: 100, color: ThemeService.getButtonColor(context)),
              const SizedBox(height: 20),
              Text(
                '发射计数器',
                style: TextStyle(
                  fontSize: 28,
                  fontWeight: FontWeight.bold,
                  color: ThemeService.getTextColor(context),
                ),
              ),
              const SizedBox(height: 30),
              TextFormField(
                controller: _usernameController,
                decoration: InputDecoration(
                  labelText: '用户名',
                  prefixIcon: Icon(Icons.person),
                  border: UnderlineInputBorder(),
                ),
                validator: (value) {
                  if (value == null || value.isEmpty) {
                    return '请输入用户名';
                  }
                  return null;
                },
              ),
              // 密码输入字段
              TextFormField(
                controller: _passwordController,
                decoration: InputDecoration(
                  labelText: '密码',
                  prefixIcon: Icon(Icons.lock),
                  border: UnderlineInputBorder(),
                ),
                obscureText: true,
                validator: (value) {
                  if (value == null || value.isEmpty) {
                    return '请输入密码';
                  }
                  return null;
                },
              ),
              const SizedBox(height: 20),
              if (_errorMessage != null)
                Text(
                  _errorMessage!,
                  style: TextStyle(color: Colors.red, fontSize: 16),
                ),
              const SizedBox(height: 10),
              _isLoading
                  ? CircularProgressIndicator()
                  : SizedBox(
                      width: double.infinity,
                      height: 50,
                      child: ElevatedButton(
                        onPressed: _submit,
                        style: ElevatedButton.styleFrom(
                          backgroundColor: ThemeService.getButtonColor(context),
                        ),
                        child: Text(
                          '开始发射',
                          style: TextStyle(
                            fontSize: 18,
                            color: Colors.black,
                          ),
                        ),
                      ),
                    ),

            ],
          ),
        ),
      ),
    );
  }

  /// 表单提交处理方法
  void _submit() async {
    // 验证表单输入有效性
    if (_formKey.currentState!.validate()) {
      setState(() {
        _isLoading = true;
        _errorMessage = null;
      });

      final authService = AuthService(widget.storageService);
      final error = await authService.unifiedAuth(
        _usernameController.text,
        _passwordController.text
      );

      setState(() => _isLoading = false); // 结束加载状态
      
      // 处理认证结果
      if (error != null) {
        setState(() => _errorMessage = error); // 显示错误信息
      } else {
        Navigator.of(context).pushReplacement(
          MaterialPageRoute(builder: (_) => App(storageService: widget.storageService)),
        );
      }
    }
  }
}