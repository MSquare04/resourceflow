import { formatUtcDateTime } from "../utils/datetime";

export function DashboardPage(): JSX.Element {
  return (
    <section>
      <h2>Dashboard</h2>
      <p>Frontend foundation is ready. Start filling business widgets here.</p>
      <p className="muted">UTC sample: {formatUtcDateTime(new Date().toISOString())}</p>
    </section>
  );
}