import type { ReactNode } from "react";

interface PageHeaderProps {
  title: string;
  description?: string;
  actions?: ReactNode;
}

export function PageHeader({ title, description, actions }: PageHeaderProps): JSX.Element {
  return (
    <div className="page-header">
      <div className="page-header__content">
        <h2 className="page-header__title">{title}</h2>
        {description ? <p className="page-header__description muted">{description}</p> : null}
      </div>
      {actions ? <div className="page-header__actions">{actions}</div> : null}
    </div>
  );
}
