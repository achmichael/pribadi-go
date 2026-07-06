import { useState, useEffect } from 'react';
import { api } from '../../lib/api';
import { useParams, Link } from 'react-router-dom';
import Form from '@rjsf/core';
import validator from '@rjsf/validator-ajv8';

interface Record {
  id: string;
  data_json: string;
  created_at: string;
}

const EntityRecords = () => {
  const { id } = useParams<{ id: string }>(); // Using entity_name actually as per the route
  const [schema, setSchema] = useState<any>(null);
  const [records, setRecords] = useState<Record[]>([]);
  const [loading, setLoading] = useState(true);
  const [showForm, setShowForm] = useState(false);
  const [formData, setFormData] = useState<any>({});

  useEffect(() => {
    fetchData();
  }, [id]);

  const fetchData = async () => {
    try {
      setLoading(true);
      // We expect id to be entity_name
      const [schemaRes, recordsRes] = await Promise.all([
        api.get(`/entities/schemas/${id}`),
        api.get(`/entities/schemas/${id}/records`)
      ]);
      setSchema(JSON.parse(schemaRes.data.fields_json));
      setRecords(recordsRes.data);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  const handleSubmit = async (e: any) => {
    try {
      await api.post(`/entities/schemas/${id}/records`, {
        data_json: JSON.stringify(e.formData)
      });
      setShowForm(false);
      setFormData({});
      fetchData();
    } catch (err: any) {
      alert('Failed to save record: ' + (err.response?.data?.error || err.message));
    }
  };

  if (loading) return <div>Loading...</div>;
  if (!schema) return <div>Schema not found</div>;

  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <div>
          <Link to="/entities" className="text-blue-600 hover:underline text-sm mb-2 inline-block">&larr; Back to Schemas</Link>
          <h1 className="text-2xl font-bold">Records for {id}</h1>
        </div>
        <button
          onClick={() => setShowForm(!showForm)}
          className="bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700 transition-colors"
        >
          {showForm ? 'Cancel' : 'Add Record'}
        </button>
      </div>

      {showForm && (
        <div className="bg-white p-6 rounded-lg shadow-sm border mb-6 max-w-3xl">
          <Form
            schema={schema}
            validator={validator}
            formData={formData}
            onChange={(e) => setFormData(e.formData)}
            onSubmit={handleSubmit}
            className="rjsf-tailwind"
          />
        </div>
      )}

      <div className="bg-white rounded-lg shadow-sm border overflow-hidden">
        <table className="w-full text-left text-sm">
          <thead className="bg-slate-50 border-b">
            <tr>
              <th className="px-6 py-3 font-medium text-slate-500">ID</th>
              <th className="px-6 py-3 font-medium text-slate-500">Data</th>
              <th className="px-6 py-3 font-medium text-slate-500">Created At</th>
            </tr>
          </thead>
          <tbody className="divide-y">
            {records.map((r) => {
              const data = JSON.parse(r.data_json);
              return (
                <tr key={r.id} className="hover:bg-slate-50">
                  <td className="px-6 py-4 text-slate-500 font-mono text-xs">{r.id.split('-')[0]}...</td>
                  <td className="px-6 py-4">
                    <pre className="bg-slate-100 p-2 rounded text-xs overflow-x-auto max-w-lg">
                      {JSON.stringify(data, null, 2)}
                    </pre>
                  </td>
                  <td className="px-6 py-4 text-slate-500">{new Date(r.created_at).toLocaleString()}</td>
                </tr>
              );
            })}
            {records.length === 0 && (
              <tr>
                <td colSpan={3} className="px-6 py-8 text-center text-slate-500">
                  No records found.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
};

export default EntityRecords;
