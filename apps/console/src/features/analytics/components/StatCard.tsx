import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  Sparkline,
  statDeltaVariants,
  cn,
} from "@qeetrix/ui";

interface StatCardProps {
  title: string;
  value: string | number;
  delta?: number;
  trend?: "up" | "down" | "flat";
  sparkData?: number[];
  className?: string;
}

export function StatCard({ title, value, delta, trend = "flat", sparkData, className }: StatCardProps) {
  return (
    <Card className={className}>
      <CardHeader className="pb-2">
        <CardTitle className="text-sm font-medium text-muted-foreground">{title}</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="flex items-end justify-between gap-4">
          <div>
            <p className="text-2xl font-bold">{value}</p>
            {delta !== undefined && (
              <p className={cn("mt-1 text-xs", statDeltaVariants({ trend }))}>
                {trend === "up" ? "▲" : trend === "down" ? "▼" : "–"} {Math.abs(delta)}% vs last period
              </p>
            )}
          </div>
          {sparkData && sparkData.length > 0 && (
            <Sparkline data={sparkData} type="area" height={40} className="w-24 shrink-0" />
          )}
        </div>
      </CardContent>
    </Card>
  );
}
