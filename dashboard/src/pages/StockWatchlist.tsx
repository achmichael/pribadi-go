import { useState, useEffect } from "react";
import { api } from "../lib/api";

interface Stock {
  id: string;
  ticker: string;
  threshold_value: number;
  alert_condition: string;
  is_active: boolean;
}

const StockWatchlist = () => {
  const [stocks, setStocks] = useState<Stock[]>([]);
  const [loading, setLoading] = useState(true);
  const [ticker, setTicker] = useState("");
  const [targetPrice, setTargetPrice] = useState("");
  const [condition, setCondition] = useState("above");

  useEffect(() => {
    fetchStocks();
  }, []);

  const fetchStocks = async () => {
    try {
      const res = await api.get("/stocks");
      setStocks(res.data || []);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  const handleAddStock = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await api.post("/stocks", {
        ticker: ticker.toUpperCase(),
        exchange: "IDX",
        threshold_value: parseFloat(targetPrice) || 0,
        alert_condition: condition,
        is_active: true,
      });
      setTicker("");
      setTargetPrice("");
      fetchStocks();
    } catch (err: any) {
      alert("Failed to add stock: " + err.message);
    }
  };

  const toggleActive = async (stock: Stock) => {
    try {
      await api.put(`/stocks/${stock.id}`, {
        ...stock,
        is_active: !stock.is_active,
      });
      fetchStocks();
    } catch {
      alert("Failed to toggle stock status");
    }
  };

  if (loading) return <div>Loading...</div>;

  return (
    <div>
      <h1 className="font-sora text-2xl font-bold mb-6">Stock Watchlist</h1>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <div className="md:col-span-1">
          <div className="bg-surface p-6 rounded-lg shadow-sm border">
            <h2 className="font-sora font-semibold text-lg mb-4">Add Stock</h2>
            <form onSubmit={handleAddStock} className="space-y-4">
              <div>
                <label className="block text-sm font-medium mb-1">
                  Ticker Symbol
                </label>
                <input
                  type="text"
                  value={ticker}
                  onChange={(e) => setTicker(e.target.value)}
                  placeholder="e.g. AAPL"
                  required
                  className="w-full border rounded-md px-3 py-2"
                />
              </div>
              <div>
                <label className="block text-sm font-medium mb-1">
                  Target Price
                </label>
                <input
                  type="number"
                  step="0.01"
                  value={targetPrice}
                  onChange={(e) => setTargetPrice(e.target.value)}
                  placeholder="e.g. 150.00"
                  className="w-full border rounded-md px-3 py-2"
                />
              </div>
              <div>
                <label className="block text-sm font-medium mb-1">
                  Condition
                </label>
                <select
                  value={condition}
                  onChange={(e) => setCondition(e.target.value)}
                  className="w-full border rounded-md px-3 py-2"
                >
                  <option value="above">Above Target</option>
                  <option value="below">Below Target</option>
                </select>
              </div>
              <button
                type="submit"
                className="w-full bg-blue-600 text-white rounded-md py-2 font-medium hover:bg-blue-700"
              >
                Add Watchlist
              </button>
            </form>
          </div>
        </div>

        <div className="md:col-span-2">
          <div className="bg-surface rounded-lg shadow-sm border overflow-hidden">
            <table className="w-full text-left text-sm">
              <thead className="bg-slate-50 border-b">
                <tr>
                  <th className="px-6 py-3 font-medium text-slate-500">
                    Ticker
                  </th>
                  <th className="px-6 py-3 font-medium text-slate-500">
                    Target
                  </th>
                  <th className="px-6 py-3 font-medium text-slate-500">
                    Condition
                  </th>
                  <th className="px-6 py-3 font-medium text-slate-500">
                    Status
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y">
                {stocks.map((s) => (
                  <tr key={s.id} className="hover:bg-slate-50">
                    <td className="px-6 py-4 font-bold text-slate-900">
                      {s.ticker}
                    </td>
                    <td className="px-6 py-4">
                      {s.threshold_value ? `$${s.threshold_value}` : "-"}
                    </td>
                    <td className="px-6 py-4">{s.alert_condition}</td>
                    <td className="px-6 py-4">
                      <button
                        onClick={() => toggleActive(s)}
                        className={`px-3 py-1 rounded-full text-xs font-medium ${
                          s.is_active
                            ? "bg-green-100 text-green-800"
                            : "bg-slate-100 text-slate-800"
                        }`}
                      >
                        {s.is_active ? "Active" : "Paused"}
                      </button>
                    </td>
                  </tr>
                ))}
                {stocks.length === 0 && (
                  <tr>
                    <td
                      colSpan={4}
                      className="px-6 py-8 text-center text-slate-500"
                    >
                      Watchlist is empty.
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>
  );
};

export default StockWatchlist;
