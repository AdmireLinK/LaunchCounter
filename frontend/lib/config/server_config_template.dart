class ServerConfig {
  static const String domain = "your_domain.com:port"; //在此处输入后端ip或域名:端口
  static const bool useTLS = true; //是否使用tls

  static String get _httpProtocol => useTLS ? "https://" : "http://";
  static String get _wsProtocol => useTLS ? "wss://" : "ws://";

  static String getUnifiedAuthUrl() => "${_httpProtocol}$domain/auth";
  static String getSyncUrl() => "${_httpProtocol}$domain/sync";
  static String getWsUrl() => "${_wsProtocol}$domain/ws";
}