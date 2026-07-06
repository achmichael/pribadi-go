import { useState, useEffect } from 'react';
import { api } from '../lib/api';

interface Config {
  id: string;
  config_key: string;
  value_json: string;
  description: string;
}

const AgentConfig = () => {
  const [configs, setConfigs] = useState<Config[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    fetchConfigs();
  }, []);

  const fetchConfigs = async () => {
    try {
      const res = await api.get('/config');
      setConfigs(res.data);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  const handleSave = async (config: Config, newValue: string) => {
    try {
      setSaving(true);
      await api.post('/config', {
        config_key: config.config_key,
        value_json: newValue,
        description: config.description,
      });
      await fetchConfigs();
    } catch (err) {
      console.error(err);
      alert('Failed to save configuration');
    } finally {
      setSaving(false);
    }
  };

  if (loading) return <div>Loading...</div>;

  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold">Agent Configuration</h1>
        {saving && <span className="text-sm text-blue-600 bg-blue-50 px-2 py-1 rounded">Saving...</span>}
      </div>
      <div className="space-y-6">
        {configs.map((config) => (
          <div key={config.id} className="bg-white p-6 rounded-lg shadow-sm border">
            <div className="mb-4">
              <h3 className="font-semibold text-lg">{config.config_key}</h3>
              <p className="text-slate-500 text-sm">{config.description}</p>
            </div>
            <textarea
              className="w-full border rounded-md p-3 font-mono text-sm focus:ring-2 focus:ring-blue-500 focus:outline-none"
              rows={4}
              defaultValue={config.value_json}
              onBlur={(e) => {
                if (e.target.value !== config.value_json) {
                  handleSave(config, e.target.value);
                }
              }}
            />
          </div>
        ))}
        {configs.length === 0 && (
          <div className="text-center text-slate-500 py-10">
            No configurations found. Add them via API or Database.
          </div>
        )}
      </div>
    </div>
  );
};

export default AgentConfig;
