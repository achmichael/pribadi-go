import { useState } from 'react';
import { api } from '../../lib/api';
import { useNavigate } from 'react-router-dom';

const SchemaBuilder = () => {
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [fieldsJSON, setFieldsJSON] = useState('{\n  "$schema": "https://json-schema.org/draft/2020-12/schema",\n  "type": "object",\n  "properties": {\n    "example": {\n      "type": "string"\n    }\n  }\n}');
  const navigate = useNavigate();

  const handleSave = async () => {
    try {
      // Basic validate JSON
      JSON.parse(fieldsJSON);

      await api.post('/entities/schemas', {
        entity_name: name,
        description: description,
        fields_json: fieldsJSON,
      });
      navigate('/entities');
    } catch (err: any) {
      alert('Failed to save schema: ' + err.message);
    }
  };

  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">Create Entity Schema</h1>
      <div className="bg-white p-6 rounded-lg shadow-sm border max-w-2xl">
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-1">Entity Name</label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value.toLowerCase().replace(/\s+/g, '_'))}
              placeholder="e.g. customer_feedback"
              className="w-full border rounded-md px-3 py-2 focus:ring-2 focus:ring-blue-500"
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">Description</label>
            <input
              type="text"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              className="w-full border rounded-md px-3 py-2 focus:ring-2 focus:ring-blue-500"
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">JSON Schema Definition</label>
            <textarea
              rows={15}
              value={fieldsJSON}
              onChange={(e) => setFieldsJSON(e.target.value)}
              className="w-full border rounded-md px-3 py-2 font-mono text-sm focus:ring-2 focus:ring-blue-500"
            />
          </div>
          <button
            onClick={handleSave}
            className="w-full bg-blue-600 text-white rounded-md py-2 font-medium hover:bg-blue-700 transition-colors"
          >
            Save Schema
          </button>
        </div>
      </div>
    </div>
  );
};

export default SchemaBuilder;
