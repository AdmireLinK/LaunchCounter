class ServerConfig {
  static const String domain = "your_domain.com:port";
  static const bool useTLS = true;

  static String get _httpProtocol => useTLS ? "https://" : "http://";
  static String get _wsProtocol => useTLS ? "wss://" : "ws://";

  static String getUnifiedAuthUrl() => "${_httpProtocol}$domain/auth";
  static String getSyncUrl() => "${_httpProtocol}$domain/sync";
  static String getWsUrl() => "${_wsProtocol}$domain/ws";
}