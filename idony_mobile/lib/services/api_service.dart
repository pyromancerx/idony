import 'dart:convert';
import 'package:http/http.dart' as http;

class ApiService {
  String baseUrl;
  String apiKey;

  ApiService({required this.baseUrl, required this.apiKey});

  Map<String, String> get _headers => {
    'Content-Type': 'application/json',
    'X-API-Key': apiKey,
  };

  Future<Map<String, dynamic>> getSchemas() async {
    final response = await http.get(Uri.parse('$baseUrl/ui/schemas'), headers: _headers);
    if (response.statusCode == 200) {
      return json.decode(response.body);
    }
    throw Exception('Failed to load UI schemas');
  }

  Future<String> sendChat(String text, {List<String>? images}) async {
    final response = await http.post(
      Uri.parse('$baseUrl/chat'),
      headers: _headers,
      body: json.encode({
        'text': text,
        if (images != null) 'images': images,
      }),
    );
    if (response.statusCode == 200) {
      return json.decode(response.body)['response'];
    }
    throw Exception('Failed to send message');
  }
}
