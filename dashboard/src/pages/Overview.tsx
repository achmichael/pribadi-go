const Overview = () => {
  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">Overview</h1>
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <div className="bg-white p-6 rounded-lg shadow-sm border">
          <h3 className="text-slate-500 text-sm font-medium">Active Cron Jobs</h3>
          <p className="text-3xl font-bold mt-2">--</p>
        </div>
        <div className="bg-white p-6 rounded-lg shadow-sm border">
          <h3 className="text-slate-500 text-sm font-medium">Custom Entities</h3>
          <p className="text-3xl font-bold mt-2">--</p>
        </div>
        <div className="bg-white p-6 rounded-lg shadow-sm border">
          <h3 className="text-slate-500 text-sm font-medium">Stock Watchlist</h3>
          <p className="text-3xl font-bold mt-2">--</p>
        </div>
      </div>
    </div>
  );
};

export default Overview;
