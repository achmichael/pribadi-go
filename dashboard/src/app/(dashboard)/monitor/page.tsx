import { MonitorList } from '@/components/monitor/MonitorList';

export default function MonitorPage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Data Monitors</h1>
        <p className="text-muted-foreground">
          Watch external APIs or websites for specific conditions to trigger notifications.
        </p>
      </div>
      <MonitorList />
    </div>
  );
}