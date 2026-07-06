import { useState, useEffect } from 'react';
import { api } from '../../lib/api';
import { Link } from 'react-router-dom';

interface CronJob {
  id: string;
  job_type: string;
  schedule_cron: string;
  is_active: boolean;
  target_whatsapp_jid: string;
}

const JobList = () => {
  const [jobs, setJobs] = useState<CronJob[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchJobs();
  }, []);

  const fetchJobs = async () => {
    try {
      const res = await api.get('/cron');
      setJobs(res.data);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  const toggleActive = async (job: CronJob) => {
    try {
      await api.put(`/cron/${job.id}`, {
        ...job,
        is_active: !job.is_active,
      });
      fetchJobs();
    } catch (err) {
      alert('Failed to toggle job status');
    }
  };

  if (loading) return <div>Loading...</div>;

  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold">Cron Jobs</h1>
        <Link
          to="/cron/new"
          className="bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700 transition-colors"
        >
          Create Job
        </Link>
      </div>

      <div className="bg-white rounded-lg shadow-sm border overflow-hidden">
        <table className="w-full text-left">
          <thead className="bg-slate-50 border-b">
            <tr>
              <th className="px-6 py-3 font-medium text-slate-500">Type</th>
              <th className="px-6 py-3 font-medium text-slate-500">Schedule</th>
              <th className="px-6 py-3 font-medium text-slate-500">Target JID</th>
              <th className="px-6 py-3 font-medium text-slate-500">Status</th>
              <th className="px-6 py-3 font-medium text-slate-500">Actions</th>
            </tr>
          </thead>
          <tbody className="divide-y">
            {jobs.map((j) => (
              <tr key={j.id} className="hover:bg-slate-50 transition-colors">
                <td className="px-6 py-4 font-medium text-slate-900">{j.job_type}</td>
                <td className="px-6 py-4 font-mono text-sm">{j.schedule_cron}</td>
                <td className="px-6 py-4 text-slate-500">{j.target_whatsapp_jid || '-'}</td>
                <td className="px-6 py-4">
                  <button
                    onClick={() => toggleActive(j)}
                    className={`px-3 py-1 rounded-full text-xs font-medium ${
                      j.is_active ? 'bg-green-100 text-green-800' : 'bg-slate-100 text-slate-800'
                    }`}
                  >
                    {j.is_active ? 'Active' : 'Paused'}
                  </button>
                </td>
                <td className="px-6 py-4">
                  <Link to={`/cron/${j.id}/edit`} className="text-blue-600 hover:underline">
                    Edit
                  </Link>
                </td>
              </tr>
            ))}
            {jobs.length === 0 && (
              <tr>
                <td colSpan={5} className="px-6 py-8 text-center text-slate-500">
                  No cron jobs found. Create one to get started.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
};

export default JobList;
