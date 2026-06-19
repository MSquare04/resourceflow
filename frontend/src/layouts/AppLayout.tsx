import { NavLink, Outlet, useNavigate } from "react-router-dom";

import { logout } from "../utils/auth";

const links = [
  { to: "/", label: "Dashboard", end: true },
  { to: "/resources", label: "Resources" },
  { to: "/my-bookings", label: "My Bookings" },
  { to: "/users", label: "Users" },
  { to: "/booking-rules", label: "Booking Rules" },
];

export function AppLayout(): JSX.Element {
  const navigate = useNavigate();

  const handleLogout = (): void => {
    logout();
    navigate("/login", { replace: true });
  };

  return (
    <div className="app-shell">
      <aside className="sidebar">
        <div className="brand">ResourceFlow</div>
        <nav className="menu">
          {links.map((link) => (
            <NavLink
              key={link.to}
              to={link.to}
              end={link.end}
              className={({ isActive }) => `menu-link ${isActive ? "active" : ""}`}
            >
              {link.label}
            </NavLink>
          ))}
        </nav>
      </aside>
      <div className="main-area">
        <header className="header">
          <h1 className="title">Workspace</h1>
          <button type="button" className="btn btn-secondary" onClick={handleLogout}>
            Logout
          </button>
        </header>
        <main className="content">
          <Outlet />
        </main>
      </div>
    </div>
  );
}