import { useState, useEffect } from 'react';
import { api } from '../../lib/api';
import { Link } from 'react-router-dom';

interface Schema {
  id: string;
  entity_name: string;
  description: string;
  created_at: string;
}

const SchemaList = () => {
  const [schemas, setSchemas] = useState<Schema[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchSchemas();
  }, []);

  const fetchSchemas = async () => {
    try {
      const res = await api.get('/entities/schemas');
      setSchemas(res.data);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  if (loading) return <div>Loading...</div>;

  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold">Custom Entities</h1>
        <Link
          to="/entities/new"
          className="bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700 transition-colors"
        >
          Create Schema
        </Link>
      </div>

      <div className="bg-white rounded-lg shadow-sm border overflow-hidden">
        <table className="w-full text-left">
          <thead className="bg-slate-50 border-b">
            <tr>
              <th className="px-6 py-3 font-medium text-slate-500">Name</th>
              <th className="px-6 py-3 font-medium text-slate-500">Description</th>
              <th className="px-6 py-3 font-medium text-slate-500">Actions</th>
            </tr>
          </thead>
          <tbody className="divide-y">
            {schemas.map((s) => (
              <tr key={s.id} className="hover:bg-slate-50 transition-colors">
                <td className="px-6 py-4 font-medium text-slate-900">{s.entity_name}</td>
                <td className="px-6 py-4 text-slate-500">{s.description}</td>
                <td className="px-6 py-4 space-x-4">
                  <Link to={`/entities/${s.entity_name}/records`} className="text-blue-600 hover:text-blue-800">
                    View Records
                  </Link>
                </td>
              </tr>
            ))}
            {schemas.length === 0 && (
              <tr>
                <td colSpan={3} className="px-6 py-8 text-center text-slate-500">
                  No schemas found. Create one to get started.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
};

export default SchemaList;
