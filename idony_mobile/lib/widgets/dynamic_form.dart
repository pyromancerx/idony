import 'package:flutter/material.dart';

class DynamicToolForm extends StatefulWidget {
  final Map<String, dynamic> schema;
  final Function(Map<String, dynamic>) onSubmit;

  const DynamicToolForm({super.key, required this.schema, required this.onSubmit});

  @override
  State<DynamicToolForm> createState() => _DynamicToolFormState();
}

class _DynamicToolFormState extends State<DynamicToolForm> {
  final Map<String, dynamic> _formData = {};
  final _formKey = GlobalKey<FormState>();

  Widget _buildField(Map<String, dynamic> field) {
    final String type = field['type'] ?? 'string';
    final String name = field['name'];
    final String label = field['label'] ?? name;

    if (type == 'choice') {
      final List<dynamic> options = field['options'] ?? [];
      return DropdownButtonFormField<String>(
        decoration: InputDecoration(labelText: label),
        items: options.map((o) => DropdownMenuItem(value: o.toString(), child: Text(o.toString()))).toList(),
        onChanged: (val) => _formData[name] = val,
      );
    }

    return TextFormField(
      decoration: InputDecoration(
        labelText: label,
        hintText: field['hint'],
      ),
      maxLines: type == 'longtext' ? 5 : 1,
      onSaved: (val) => _formData[name] = val,
    );
  }

  @override
  Widget build(BuildContext context) {
    // Some schemas might have 'actions' (like SubAgent), some might just have 'fields'
    final List<dynamic> actions = widget.schema['actions'] as List<dynamic>? ?? [];
    final List<dynamic> fields = widget.schema['fields'] as List<dynamic>? ?? [];
    
    if (actions.isNotEmpty) {
      return DefaultTabController(
        length: actions.length,
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(widget.schema['title'] ?? 'Tool Action', style: Theme.of(context).textTheme.headlineSmall),
            TabBar(
              isScrollable: true,
              tabs: actions.map((a) => Tab(text: a['label'] ?? a['name'])).toList(),
            ),
            SizedBox(
              height: 300, // Fixed height for tab content in bottom sheet
              child: TabBarView(
                children: actions.map((a) {
                  final actionFields = a['fields'] as List<dynamic>? ?? [];
                  return _ActionForm(
                    actionName: a['name'],
                    fields: actionFields,
                    onSubmit: (data) => widget.onSubmit({'action': a['name'], ...data}),
                  );
                }).toList(),
              ),
            ),
          ],
        ),
      );
    }

    return Form(
      key: _formKey,
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Text(widget.schema['title'] ?? 'Tool Action', style: Theme.of(context).textTheme.headlineSmall),
          const SizedBox(height: 16),
          ...fields.map((f) => Padding(
            padding: const EdgeInsets.only(bottom: 12),
            child: _buildField(f),
          )),
          const SizedBox(height: 20),
          ElevatedButton(
            onPressed: () {
              _formKey.currentState!.save();
              widget.onSubmit(_formData);
            },
            child: const Text('Execute Tool'),
          )
        ],
      ),
    );
  }
}

class _ActionForm extends StatefulWidget {
  final String actionName;
  final List<dynamic> fields;
  final Function(Map<String, dynamic>) onSubmit;

  const _ActionForm({required this.actionName, required this.fields, required this.onSubmit});

  @override
  State<_ActionForm> createState() => _ActionFormState();
}

class _ActionFormState extends State<_ActionForm> {
  final Map<String, dynamic> _formData = {};
  final _formKey = GlobalKey<FormState>();

  @override
  Widget build(BuildContext context) {
    return SingleChildScrollView(
      child: Form(
        key: _formKey,
        child: Padding(
          padding: const EdgeInsets.all(16.0),
          child: Column(
            children: [
              ...widget.fields.map((f) {
                final String type = f['type'] ?? 'string';
                final String name = f['name'];
                final String label = f['label'] ?? name;

                if (type == 'choice') {
                  final List<dynamic> options = f['options'] ?? [];
                  return DropdownButtonFormField<String>(
                    decoration: InputDecoration(labelText: label),
                    items: options.map((o) => DropdownMenuItem(value: o.toString(), child: Text(o.toString()))).toList(),
                    onChanged: (val) => _formData[name] = val,
                  );
                }

                return TextFormField(
                  decoration: InputDecoration(labelText: label, hintText: f['hint']),
                  maxLines: type == 'longtext' ? 3 : 1,
                  onSaved: (val) => _formData[name] = val,
                );
              }),
              const SizedBox(height: 20),
              ElevatedButton(
                onPressed: () {
                  _formKey.currentState!.save();
                  widget.onSubmit(_formData);
                },
                child: Text('Run ${widget.actionName}'),
              )
            ],
          ),
        ),
      ),
    );
  }
}
