import 'dart:convert';
import 'package:flutter/material.dart';
import 'package:flutter_markdown/flutter_markdown.dart';
import 'package:image_picker/image_picker.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'services/api_service.dart';
import 'widgets/dynamic_form.dart';

void main() {
  runApp(const IdonyApp());
}

class IdonyApp extends StatelessWidget {
  const IdonyApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Idony Native',
      debugShowCheckedModeBanner: false,
      theme: ThemeData(
        colorScheme: ColorScheme.fromSeed(
          seedColor: Colors.cyan,
          brightness: Brightness.dark,
        ),
        useMaterial3: true,
      ),
      home: const SecureGate(),
    );
  }
}

class SecureGate extends StatefulWidget {
  const SecureGate({super.key});

  @override
  State<SecureGate> createState() => _SecureGateState();
}

class _SecureGateState extends State<SecureGate> {
  final _storage = const FlutterSecureStorage();
  bool _isConfigured = false;
  String? _url;
  String? _key;

  @override
  void initState() {
    super.initState();
    _checkConfig();
  }

  void _checkConfig() async {
    _url = await _storage.read(key: 'server_url');
    _key = await _storage.read(key: 'api_key');
    if (_url != null && _key != null) {
      setState(() => _isConfigured = true);
    }
  }

  @override
  Widget build(BuildContext context) {
    if (!_isConfigured) {
      return SetupScreen(onComplete: (url, key) async {
        await _storage.write(key: 'server_url', value: url);
        await _storage.write(key: 'api_key', value: key);
        setState(() {
          _url = url;
          _key = key;
          _isConfigured = true;
        });
      });
    }
    return MainContainer(baseUrl: _url!, apiKey: _key!);
  }
}

class SetupScreen extends StatelessWidget {
  final Function(String, String) onComplete;
  final _urlController = TextEditingController();
  final _keyController = TextEditingController();

  SetupScreen({super.key, required this.onComplete});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: Padding(
        padding: const EdgeInsets.all(24.0),
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            const Icon(Icons.security, size: 64, color: Colors.cyan),
            const SizedBox(height: 16),
            const Text('Secure Idony Setup', style: TextStyle(fontSize: 24, fontWeight: FontWeight.bold)),
            const SizedBox(height: 32),
            TextField(
              controller: _urlController,
              decoration: const InputDecoration(labelText: 'Server URL (include https://)', border: OutlineInputBorder()),
            ),
            const SizedBox(height: 16),
            TextField(
              controller: _keyController,
              obscureText: true,
              decoration: const InputDecoration(labelText: 'Server API Key', border: OutlineInputBorder()),
            ),
            const SizedBox(height: 32),
            ElevatedButton(
              style: ElevatedButton.styleFrom(minimumSize: const Size(double.infinity, 50), backgroundColor: Colors.cyan),
              onPressed: () => onComplete(_urlController.text, _keyController.text),
              child: const Text('Connect Securely', style: TextStyle(color: Colors.black)),
            ),
          ],
        ),
      ),
    );
  }
}

class MainContainer extends StatefulWidget {
  final String baseUrl;
  final String apiKey;
  const MainContainer({super.key, required this.baseUrl, required this.apiKey});

  @override
  State<MainContainer> createState() => _MainContainerState();
}

class _MainContainerState extends State<MainContainer> {
  late ApiService _api;
  final ImagePicker _picker = ImagePicker();
  int _selectedIndex = 0;
  final List<Map<String, dynamic>> _messages = [];
  Map<String, dynamic> _schemas = {};
  List<String> _selectedImagesB64 = [];
  bool _isLoading = false;

  @override
  void initState() {
    super.initState();
    _api = ApiService(baseUrl: widget.baseUrl, apiKey: widget.apiKey);
    _loadSchemas();
  }

  void _loadSchemas() async {
    try {
      final s = await _api.getSchemas();
      setState(() => _schemas = s);
    } catch (e) {
      debugPrint('Error loading schemas: $e');
    }
  }

  void _pickImage() async {
    final XFile? image = await _picker.pickImage(source: ImageSource.gallery);
    if (image != null) {
      final bytes = await image.readAsBytes();
      setState(() {
        _selectedImagesB64.add(base64Encode(bytes));
      });
    }
  }

  void _sendMessage(String text) async {
    if (text.isEmpty && _selectedImagesB64.isEmpty) return;

    setState(() {
      _messages.add({
        'role': 'user', 
        'content': text,
        'hasImages': _selectedImagesB64.isNotEmpty
      });
      _isLoading = true;
    });

    try {
      final response = await _api.sendChat(text, images: _selectedImagesB64.isNotEmpty ? _selectedImagesB64 : null);
      setState(() {
        _messages.add({'role': 'assistant', 'content': response});
        _selectedImagesB64 = [];
      });
    } catch (e) {
      setState(() => _messages.add({'role': 'assistant', 'content': 'Error: $e'}));
    } finally {
      setState(() => _isLoading = false);
    }
  }

  void _showToolDialog(String toolName, Map<String, dynamic> schema) {
    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      shape: const RoundedRectangleBorder(borderRadius: BorderRadius.vertical(top: Radius.circular(20))),
      builder: (context) => Padding(
        padding: EdgeInsets.only(
          bottom: MediaQuery.of(context).viewInsets.bottom,
          left: 20, right: 20, top: 20,
        ),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            DynamicToolForm(
              schema: schema,
              onSubmit: (data) {
                Navigator.pop(context);
                String jsonInput = json.encode(data);
                _sendMessage('/$toolName $jsonInput');
              },
            ),
            const SizedBox(height: 20),
          ],
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Idony AI', style: TextStyle(fontWeight: FontWeight.bold, color: Colors.cyan)),
        actions: [
          IconButton(icon: const Icon(Icons.logout), onPressed: () async {
            await const FlutterSecureStorage().deleteAll();
            // In a real app, you'd navigate back to SecureGate
          }),
          IconButton(icon: const Icon(Icons.refresh), onPressed: _loadSchemas),
        ],
      ),
      body: _selectedIndex == 0 ? _buildChat() : _buildToolbox(),
      bottomNavigationBar: NavigationBar(
        selectedIndex: _selectedIndex,
        onDestinationSelected: (i) => setState(() => _selectedIndex = i),
        destinations: const [
          NavigationDestination(icon: Icon(Icons.forum_outlined), selectedIcon: Icon(Icons.forum), label: 'Chat'),
          NavigationDestination(icon: Icon(Icons.grid_view_outlined), selectedIcon: Icon(Icons.grid_view), label: 'Toolbox'),
        ],
      ),
    );
  }

  Widget _buildChat() {
    return Column(
      children: [
        Expanded(
          child: ListView.builder(
            padding: const EdgeInsets.all(12),
            itemCount: _messages.length,
            itemBuilder: (context, i) {
              final m = _messages[i];
              bool isUser = m['role'] == 'user';
              return Align(
                alignment: isUser ? Alignment.centerRight : Alignment.centerLeft,
                child: Container(
                  margin: const EdgeInsets.symmetric(vertical: 4),
                  padding: const EdgeInsets.all(12),
                  decoration: BoxDecoration(
                    color: isUser ? Colors.cyan.withOpacity(0.2) : Colors.grey[900],
                    borderRadius: BorderRadius.circular(15),
                    border: Border.all(color: isUser ? Colors.cyan : Colors.grey[800]!),
                  ),
                  constraints: BoxConstraints(maxWidth: MediaQuery.of(context).size.width * 0.8),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      if (m['hasImages'] == true) 
                        const Padding(
                          padding: EdgeInsets.only(bottom: 8.0),
                          child: Icon(Icons.image, size: 40, color: Colors.cyan),
                        ),
                      MarkdownBody(
                        data: m['content'],
                        selectable: true,
                        styleConfig: MarkdownStyleSheet(
                          p: const TextStyle(fontSize: 16),
                        ),
                      ),
                    ],
                  ),
                ),
              );
            },
          ),
        ),
        if (_isLoading) const LinearProgressIndicator(color: Colors.cyan, backgroundColor: Colors.transparent),
        if (_selectedImagesB64.isNotEmpty)
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 8.0),
            child: Row(
              children: [
                const Icon(Icons.attach_file, color: Colors.cyan),
                Text(' ${_selectedImagesB64.length} image(s) attached', style: const TextStyle(color: Colors.cyan)),
                IconButton(icon: const Icon(Icons.close, size: 16), onPressed: () => setState(() => _selectedImagesB64 = []))
              ],
            ),
          ),
        _buildInputArea(),
      ],
    );
  }

  Widget _buildInputArea() {
    final TextEditingController controller = TextEditingController();
    return Padding(
      padding: const EdgeInsets.all(12.0),
      child: Row(
        children: [
          IconButton(icon: const Icon(Icons.add_a_photo_outlined), onPressed: _pickImage),
          Expanded(
            child: TextField(
              controller: controller,
              decoration: InputDecoration(
                hintText: 'Message Idony...',
                border: OutlineInputBorder(borderRadius: BorderRadius.circular(25)),
                contentPadding: const EdgeInsets.symmetric(horizontal: 20),
              ),
              onSubmitted: (val) {
                _sendMessage(val);
                controller.clear();
              },
            ),
          ),
          const SizedBox(width: 8),
          CircleAvatar(
            backgroundColor: Colors.cyan,
            child: IconButton(
              icon: const Icon(Icons.send, color: Colors.black),
              onPressed: () {
                _sendMessage(controller.text);
                controller.clear();
              },
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildToolbox() {
    return GridView.builder(
      padding: const EdgeInsets.all(16),
      gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
        crossAxisCount: 2,
        crossAxisSpacing: 12,
        mainAxisSpacing: 12,
        childAspectRatio: 1.5,
      ),
      itemCount: _schemas.length,
      itemBuilder: (context, i) {
        String key = _schemas.keys.elementAt(i);
        var schema = _schemas[key];
        return InkWell(
          onTap: () => _showToolDialog(key, schema),
          child: Container(
            decoration: BoxDecoration(
              color: Colors.grey[900],
              borderRadius: BorderRadius.circular(15),
              border: Border.all(color: Colors.grey[800]!),
            ),
            child: Column(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                const Icon(Icons.settings_input_component, color: Colors.cyan),
                const SizedBox(height: 8),
                Text(key, style: const TextStyle(fontWeight: FontWeight.bold)),
                Text(schema['title'] ?? '', style: const TextStyle(fontSize: 10, color: Colors.grey)),
              ],
            ),
          ),
        );
      },
    );
  }
}
