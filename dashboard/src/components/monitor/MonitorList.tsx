"use client";

import { useEffect, useState } from 'react';
import { useMonitorStore } from '@/store/monitor';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { PlusIcon, Trash2Icon, ActivityIcon } from 'lucide-react';
import { MonitorFormDialog } from './MonitorFormDialog';

export function MonitorList() {
  const { tasks, isLoading, fetchTasks, deleteTask } = useMonitorStore();
  const [isDialogOpen, setIsDialogOpen] = useState(false);

  useEffect(() => {
    fetchTasks();
  }, [fetchTasks]);

  return (
    <div className="space-y-4">
      <div className="flex justify-between items-center">
        <h2 className="text-xl font-semibold">Active Monitors</h2>
        <Button onClick={() => setIsDialogOpen(true)}>
          <PlusIcon className="w-4 h-4 mr-2" />
          Add Monitor
        </Button>
      </div>

      {isLoading && <p>Loading monitors...</p>}
      
      {!isLoading && tasks.length === 0 && (
        <Card className="border-dashed">
          <CardContent className="flex flex-col items-center justify-center py-12">
            <ActivityIcon className="w-12 h-12 text-muted-foreground mb-4 opacity-50" />
            <p className="text-muted-foreground text-center">No monitors active.<br/>Create one to start tracking APIs or websites.</p>
          </CardContent>
        </Card>
      )}

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        {tasks.map((task) => (
          <Card key={task.id}>
            <CardHeader className="flex flex-row items-center justify-between pb-2">
              <CardTitle className="text-sm font-medium">
                {task.name}
              </CardTitle>
              <Button variant="ghost" size="icon" onClick={() => deleteTask(task.id)}>
                <Trash2Icon className="w-4 h-4 text-destructive" />
              </Button>
            </CardHeader>
            <CardContent>
              <div className="flex gap-2 mb-2">
                <Badge variant="outline">{task.source_type}</Badge>
                {task.category && <Badge variant="secondary">{task.category}</Badge>}
                <Badge variant={task.status === 'active' ? 'default' : 'destructive'}>{task.status}</Badge>
              </div>
              <p className="text-xs text-muted-foreground mt-2">
                Last checked: {task.last_checked_at ? new Date(task.last_checked_at).toLocaleString() : 'Never'}
              </p>
              <p className="text-sm font-semibold mt-1">
                Value: {task.last_value_text || task.last_value || 'N/A'}
              </p>
            </CardContent>
          </Card>
        ))}
      </div>

      <MonitorFormDialog open={isDialogOpen} onOpenChange={setIsDialogOpen} />
    </div>
  );
}