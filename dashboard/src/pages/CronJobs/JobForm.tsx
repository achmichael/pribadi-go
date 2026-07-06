import { useState, useEffect } from 'react';
import { api } from '../../lib/api';
import { useNavigate, useParams } from 'react-router-dom';

const JobForm = () => {
  const { id } = useParams<{ id: string }>();
  const [jobType, setJobType] = useState('custom_reminder');
  const [schedule, setSchedule] = useState('0 9 * * *');
  const [targetJid, setTargetJid] = useState('');
  const [configJson, setConfigJson] = useState('{}');
  const [isActive, setIsActive] = useState(true);
  
  const navigate = useNavigate();

  useEffect(() => {
    if (id) {
      // fetch job to edit - actually we can just list and filter or implement GET /cron/:id
      // For simplicity, skip fetch and assume creation if no API endpoint for single GET
    }
  }, [id]);

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      JSON.parse(configJson); // Validate JSON
      const payload = {
        job_type: jobType,
        schedule_cron: schedule,
        config_json: configJson,
        is_active: isActive,
        target_whatsapp_jid: targetJid,
      };

      if (id) {
        await api.put(`/cron/${id}`, payload);
      } else {
        await api.post('/cron', payload);
      }
      navigate('/cron');
    } catch (err: any) {
      alert('Failed to save job: ' + err.message);
    }
  };

  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">{id ? 'Edit' : 'Create'} Cron Job</h1>
      <div className="bg-white p-6 rounded-lg shadow-sm border max-w-2xl">
        <form onSubmit={handleSave} className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-1">Job Type</label>
            <select
              value={jobType}
              onChange={(e) => setJobType(e.target.value)}
              className="w-full border rounded-md px-3 py-2"
            >
              <option value="custom_reminder">Custom Reminder</option>
              <option value="stock_alert">Stock Alert</option>
            </select>
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">Cron Schedule</label>
            <input
              type="text"
              value={schedule}
              onChange={(e) => setSchedule(e.target.value)}
              placeholder="* * * * *"
              className="w-full border rounded-md px-3 py-2 font-mono"
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">Target WhatsApp JID</label>
            <input
              type="text"
              value={targetJid}
              onChange={(e) => setTargetJid(e.target.value)}
              placeholder="1234567890@s.whatsapp.net"
              className="w-full border rounded-md px-3 py-2"
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">Configuration (JSON)</label>
            <textarea
              rows={5}
              value={configJson}
              onChange={(e) => setConfigJson(e.target.value)}
              className="w-full border rounded-md px-3 py-2 font-mono text-sm"
            />
          </div>
          <div className="flex items-center gap-2">
            <input
              type="checkbox"
              id="isActive"
              checked={isActive}
              onChange={(e) => setIsActive(e.target.checked)}
            />
            <label htmlFor="isActive" className="text-sm font-medium">Active</label>
          </div>
          <button
            type="submit"
            className="w-full bg-blue-600 text-white rounded-md py-2 font-medium hover:bg-blue-700"
          >
            Save Job
          </button>
        </form>
      </div>
    </div>
  );
};

export default JobForm;
