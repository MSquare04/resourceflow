interface EmptyStateProps {
  title: string;
  description?: string;
}

export function EmptyState({ title, description }: EmptyStateProps): JSX.Element {
  return (
    <div className="feedback-card">
      <h3>{title}</h3>
      {description ? <p className="muted">{description}</p> : null}
    </div>
  );
}
