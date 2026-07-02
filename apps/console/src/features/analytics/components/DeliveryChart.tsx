import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  ChartContainer,
  ChartLegend,
  ChartLegendContent,
  ChartTooltip,
  ChartTooltipContent,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  type ChartConfig,
} from "@qeetrix/ui";
import { useState } from "react";
import { Area, AreaChart, CartesianGrid, XAxis, YAxis } from "recharts";

import { useDeliveryStats } from "../hooks/useDeliveryStats";

const chartConfig: ChartConfig = {
  sent:       { label: "Sent",       color: "hsl(var(--chart-1))" },
  failed:     { label: "Failed",     color: "hsl(var(--chart-2))" },
  suppressed: { label: "Suppressed", color: "hsl(var(--chart-3))" },
};

const CHANNELS = ["all", "email", "sms", "whatsapp", "inapp", "webhook"] as const;

export function DeliveryChart() {
  const [channel, setChannel] = useState<string>("all");
  const { data = [], isLoading } = useDeliveryStats(channel === "all" ? {} : { channel });

  return (
    <Card>
      <CardHeader className="flex flex-row items-start justify-between gap-4">
        <div>
          <CardTitle>Delivery overview</CardTitle>
          <CardDescription>Sent, failed, and suppressed by time bucket</CardDescription>
        </div>
        <Select value={channel} onValueChange={setChannel}>
          <SelectTrigger className="w-36">
            <SelectValue placeholder="Channel" />
          </SelectTrigger>
          <SelectContent>
            {CHANNELS.map((c) => (
              <SelectItem key={c} value={c}>
                {c === "all" ? "All channels" : c.charAt(0).toUpperCase() + c.slice(1)}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <div className="flex h-48 items-center justify-center text-sm text-muted-foreground">
            Loading chart…
          </div>
        ) : (
          <ChartContainer config={chartConfig} className="h-48 w-full">
            <AreaChart data={data}>
              <CartesianGrid strokeDasharray="3 3" vertical={false} />
              <XAxis dataKey="bucket" tick={{ fontSize: 11 }} tickLine={false} axisLine={false} />
              <YAxis tick={{ fontSize: 11 }} tickLine={false} axisLine={false} width={32} />
              <ChartTooltip content={<ChartTooltipContent />} />
              <ChartLegend content={<ChartLegendContent />} />
              <Area dataKey="sent"       type="monotone" fill="var(--color-sent)"       stroke="var(--color-sent)"       fillOpacity={0.15} />
              <Area dataKey="failed"     type="monotone" fill="var(--color-failed)"     stroke="var(--color-failed)"     fillOpacity={0.15} />
              <Area dataKey="suppressed" type="monotone" fill="var(--color-suppressed)" stroke="var(--color-suppressed)" fillOpacity={0.15} />
            </AreaChart>
          </ChartContainer>
        )}
      </CardContent>
    </Card>
  );
}
