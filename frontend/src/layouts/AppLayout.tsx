import { useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { NavLink, Outlet, useLocation, useNavigate } from "react-router-dom";

import { useAuth } from "../auth/AuthContext";
import { useRoles } from "../auth/useRoles";
import { LanguageSwitcher } from "../components/LanguageSwitcher";

type ViewportMode = "mobile" | "tablet" | "desktop";

type IconName =
  | "dashboard"
  | "resources"
  | "bookings"
  | "allBookings"
  | "users"
  | "rules"
  | "collapse"
  | "expand"
  | "logout"
  | "chevronDown";

interface NavigationLink {
  to: string;
  labelKey: string;
  icon: IconName;
  end?: boolean;
  roles?: string[];
}

const links: NavigationLink[] = [
  { to: "/dashboard", labelKey: "navigation.dashboard", icon: "dashboard", end: true },
  { to: "/resources", labelKey: "navigation.resources", icon: "resources" },
  { to: "/my-bookings", labelKey: "navigation.myBookings", icon: "bookings" },
  { to: "/bookings", labelKey: "navigation.bookings", icon: "allBookings", roles: ["admin", "manager"] },
  { to: "/users", labelKey: "navigation.users", icon: "users", roles: ["admin"] },
  { to: "/booking-rules", labelKey: "navigation.bookingRules", icon: "rules", roles: ["admin"] },
];

function AppIcon({ name }: { name: IconName }): JSX.Element {
  const commonProps = {
    viewBox: "0 0 24 24",
    fill: "none",
    xmlns: "http://www.w3.org/2000/svg",
    stroke: "currentColor",
    strokeWidth: 1.8,
    strokeLinecap: "round" as const,
    strokeLinejoin: "round" as const,
    className: "app-icon",
    "aria-hidden": true,
  };

  switch (name) {
    case "dashboard":
      return (
        <svg {...commonProps}>
          <path d="M4 13.5h7.5V20H4z" />
          <path d="M12.5 4H20v8.5h-7.5z" />
          <path d="M12.5 13.5H20V20h-7.5z" />
          <path d="M4 4h7.5v7.5H4z" />
        </svg>
      );
    case "resources":
      return (
        <svg {...commonProps}>
          <path d="M4 7.5 12 4l8 3.5-8 3.5-8-3.5Z" />
          <path d="M4 12l8 3.5 8-3.5" />
          <path d="M4 16.5 12 20l8-3.5" />
        </svg>
      );
    case "bookings":
      return (
        <svg {...commonProps}>
          <path d="M7 3v4" />
          <path d="M17 3v4" />
          <rect x="4" y="5.5" width="16" height="14.5" rx="2" />
          <path d="M4 10h16" />
          <path d="M8 14h3" />
          <path d="M13 14h3" />
        </svg>
      );
    case "allBookings":
      return (
        <svg {...commonProps}>
          <path d="M7 3v4" />
          <path d="M17 3v4" />
          <rect x="4" y="5.5" width="16" height="14.5" rx="2" />
          <path d="M4 10h16" />
          <path d="M8 14h8" />
          <path d="M8 17h5" />
        </svg>
      );
    case "users":
      return (
        <svg {...commonProps}>
          <path d="M16.5 19.5v-1.2A3.3 3.3 0 0 0 13.2 15H7.8a3.3 3.3 0 0 0-3.3 3.3v1.2" />
          <circle cx="10.5" cy="8" r="3" />
          <path d="M18 8.5a2.5 2.5 0 1 1 0 5" />
          <path d="M20 19.5v-1a3 3 0 0 0-2.2-2.9" />
        </svg>
      );
    case "rules":
      return (
        <svg {...commonProps}>
          <path d="M8 4h8" />
          <path d="M8 9h8" />
          <path d="M8 14h8" />
          <path d="M8 19h5" />
          <path d="m4.5 4.5 1 1 2-2" />
          <path d="m4.5 9.5 1 1 2-2" />
          <path d="m4.5 14.5 1 1 2-2" />
          <path d="m4.5 19.5 1 1 2-2" />
        </svg>
      );
    case "collapse":
      return (
        <svg {...commonProps}>
          <path d="m15 6-6 6 6 6" />
        </svg>
      );
    case "expand":
      return (
        <svg {...commonProps}>
          <path d="m9 6 6 6-6 6" />
        </svg>
      );
    case "logout":
      return (
        <svg {...commonProps}>
          <path d="M10 5H7a2 2 0 0 0-2 2v10a2 2 0 0 0 2 2h3" />
          <path d="M14 8l5 4-5 4" />
          <path d="M19 12H9" />
        </svg>
      );
    case "chevronDown":
      return (
        <svg {...commonProps}>
          <path d="m6 9 6 6 6-6" />
        </svg>
      );
  }
}

function getInitials(fullName?: string, email?: string): string {
  const source = (fullName || email || "").trim();
  if (!source) {
    return "RF";
  }

  if (fullName) {
    const parts = fullName.split(/\s+/).filter(Boolean).slice(0, 2);
    const initials = parts.map((part) => part[0]?.toUpperCase() ?? "").join("");
    if (initials) {
      return initials;
    }
  }

  return source.slice(0, 2).toUpperCase();
}

function getViewportMode(): ViewportMode {
  if (typeof window === "undefined") {
    return "desktop";
  }

  if (window.innerWidth <= 767) {
    return "mobile";
  }

  if (window.innerWidth <= 1099) {
    return "tablet";
  }

  return "desktop";
}

export function AppLayout(): JSX.Element {
  const navigate = useNavigate();
  const location = useLocation();
  const { t } = useTranslation();
  const { logout, user } = useAuth();
  const { hasAnyRole } = useRoles();
  const [viewportMode, setViewportMode] = useState<ViewportMode>(getViewportMode);
  const [isSidebarCollapsed, setIsSidebarCollapsed] = useState(false);
  const [isTabletSidebarOpen, setIsTabletSidebarOpen] = useState(false);
  const [isMobileDrawerOpen, setIsMobileDrawerOpen] = useState(false);
  const [isProfileOpen, setIsProfileOpen] = useState(false);
  const profileMenuRef = useRef<HTMLDivElement | null>(null);

  const visibleLinks = useMemo(
    () =>
      links.filter((link) => {
        if (!link.roles) {
          return true;
        }

        return hasAnyRole(link.roles);
      }),
    [hasAnyRole],
  );

  const localizedRoles = useMemo(
    () => user?.roles.map((role) => t(`roles.${role}`, { defaultValue: role })) ?? [],
    [t, user?.roles],
  );

  const currentPageTitle = useMemo(() => {
    const currentLink = visibleLinks.find((link) =>
      link.end ? location.pathname === link.to : location.pathname.startsWith(link.to),
    );

    if (location.pathname === "/forbidden") {
      return t("pages.forbidden.code");
    }

    return currentLink ? t(currentLink.labelKey) : t("layout.workspace");
  }, [location.pathname, t, visibleLinks]);

  const primaryIdentity = user?.full_name || user?.email || "";
  const secondaryIdentity = user?.full_name ? user.email : null;
  const userInitials = getInitials(user?.full_name, user?.email);
  const isMobile = viewportMode === "mobile";
  const isTablet = viewportMode === "tablet";
  const isDesktop = viewportMode === "desktop";
  const isOverlayOpen = isMobileDrawerOpen || (isTablet && isTabletSidebarOpen);
  const breadcrumbTitle = currentPageTitle;

  useEffect(() => {
    const handleResize = (): void => {
      setViewportMode(getViewportMode());
    };

    window.addEventListener("resize", handleResize);
    return () => {
      window.removeEventListener("resize", handleResize);
    };
  }, []);

  useEffect(() => {
    if (!isOverlayOpen && !isProfileOpen) {
      return;
    }

    const handleKeyDown = (event: KeyboardEvent): void => {
      if (event.key === "Escape") {
        setIsMobileDrawerOpen(false);
        setIsTabletSidebarOpen(false);
        setIsProfileOpen(false);
      }
    };

    document.addEventListener("keydown", handleKeyDown);
    return () => {
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, [isOverlayOpen, isProfileOpen]);

  useEffect(() => {
    if (isMobile) {
      setIsTabletSidebarOpen(false);
      return;
    }

    setIsMobileDrawerOpen(false);

    if (!isTablet) {
      setIsTabletSidebarOpen(false);
    }
  }, [isMobile, isTablet]);

  useEffect(() => {
    if (!isMobileDrawerOpen) {
      document.body.style.removeProperty("overflow");
      return;
    }

    document.body.style.overflow = "hidden";
    return () => {
      document.body.style.removeProperty("overflow");
    };
  }, [isMobileDrawerOpen]);

  useEffect(() => {
    const handlePointerDown = (event: MouseEvent): void => {
      if (!profileMenuRef.current?.contains(event.target as Node)) {
        setIsProfileOpen(false);
      }
    };

    if (!isProfileOpen) {
      return;
    }

    document.addEventListener("mousedown", handlePointerDown);
    return () => {
      document.removeEventListener("mousedown", handlePointerDown);
    };
  }, [isProfileOpen]);

  useEffect(() => {
    setIsMobileDrawerOpen(false);
    setIsTabletSidebarOpen(false);
    setIsProfileOpen(false);
  }, [location.pathname]);

  const handleLogout = (): void => {
    setIsProfileOpen(false);
    logout();
    navigate("/login", { replace: true });
  };

  const handleNavLinkClick = (): void => {
    setIsMobileDrawerOpen(false);
    setIsTabletSidebarOpen(false);
  };

  const handleSidebarToggle = (): void => {
    if (isMobile) {
      setIsMobileDrawerOpen((current) => !current);
      return;
    }

    if (isTablet) {
      setIsTabletSidebarOpen((current) => !current);
      return;
    }

    setIsSidebarCollapsed((current) => !current);
  };

  return (
    <div
      className={`app-shell layout-${viewportMode} ${isDesktop && isSidebarCollapsed ? "sidebar-collapsed" : ""} ${
        isMobileDrawerOpen ? "drawer-open" : ""
      } ${isTabletSidebarOpen ? "tablet-sidebar-open" : ""}`}
    >
      <button
        type="button"
        className={`sidebar-overlay ${isOverlayOpen ? "active" : ""}`}
        aria-label={t("layout.closeMenu")}
        onClick={() => {
          setIsMobileDrawerOpen(false);
          setIsTabletSidebarOpen(false);
        }}
      />

      <aside id="app-sidebar" className="sidebar">
        <div className="sidebar-top">
          <div className="brand">{t("common.brand")}</div>

          <button
            type="button"
            className="sidebar-toggle sidebar-panel-toggle"
            aria-label={
              (isDesktop && isSidebarCollapsed) || (isTablet && !isTabletSidebarOpen)
                ? t("layout.expandSidebar")
                : t("layout.collapseSidebar")
            }
            title={
              (isDesktop && isSidebarCollapsed) || (isTablet && !isTabletSidebarOpen)
                ? t("layout.expandSidebar")
                : t("layout.collapseSidebar")
            }
            aria-expanded={isTablet ? isTabletSidebarOpen : !isSidebarCollapsed}
            onClick={handleSidebarToggle}
          >
            <AppIcon name={(isDesktop && isSidebarCollapsed) || (isTablet && !isTabletSidebarOpen) ? "expand" : "collapse"} />
          </button>

          <button
            type="button"
            className="sidebar-toggle mobile-sidebar-close"
            aria-label={t("layout.closeMenu")}
            title={t("layout.closeMenu")}
            onClick={() => setIsMobileDrawerOpen(false)}
          >
            <AppIcon name="collapse" />
          </button>
        </div>

        <nav className="menu">
          {visibleLinks.map((link) => {
            const label = t(link.labelKey);

            return (
              <NavLink
                key={link.to}
                to={link.to}
                end={link.end}
                onClick={handleNavLinkClick}
                className={({ isActive }) => `menu-link ${isActive ? "active" : ""}`}
                title={label}
                aria-label={label}
              >
                <span className="menu-link__icon">
                  <AppIcon name={link.icon} />
                </span>
                <span className="menu-link__label">{label}</span>
              </NavLink>
            );
          })}
        </nav>
      </aside>

      <div className="main-area">
        <header className="header">
          <div className="header-start">
            {isMobile ? (
              <>
                <button
                  type="button"
                  className="sidebar-toggle mobile-sidebar-toggle"
                  aria-label={isMobileDrawerOpen ? t("layout.closeMenu") : t("layout.openMenu")}
                  title={isMobileDrawerOpen ? t("layout.closeMenu") : t("layout.openMenu")}
                  aria-expanded={isMobileDrawerOpen}
                  aria-controls="app-sidebar"
                  onClick={handleSidebarToggle}
                >
                  <AppIcon name={isMobileDrawerOpen ? "collapse" : "expand"} />
                </button>
                <h1 className="title">{currentPageTitle}</h1>
              </>
            ) : (
              <div className="breadcrumb" aria-label={t("layout.breadcrumbLabel")}>
                <span className="breadcrumb__root">{t("layout.breadcrumbRoot")}</span>
                <span className="breadcrumb__separator" aria-hidden="true">
                  /
                </span>
                <span className="breadcrumb__current">{breadcrumbTitle}</span>
              </div>
            )}
          </div>

          <div className="header-profile" ref={profileMenuRef}>
            <button
              type="button"
              className="avatar-button"
              aria-label={isProfileOpen ? t("layout.closeProfileMenu") : t("layout.openProfileMenu")}
              aria-expanded={isProfileOpen}
              aria-haspopup="menu"
              aria-controls="profile-dropdown"
              onClick={() => setIsProfileOpen((current) => !current)}
            >
              <span className="avatar-button__avatar" aria-hidden="true">
                {userInitials}
              </span>
              <span className="avatar-button__chevron" aria-hidden="true">
                <AppIcon name="chevronDown" />
              </span>
            </button>

            {isProfileOpen ? (
              <div id="profile-dropdown" className="profile-dropdown" role="menu">
                <div className="profile-dropdown__summary">
                  <span className="profile-dropdown__avatar" aria-hidden="true">
                    {userInitials}
                  </span>
                  <div className="profile-dropdown__identity">
                    <span className="profile-dropdown__primary">{primaryIdentity}</span>
                    {secondaryIdentity ? (
                      <span className="profile-dropdown__secondary">{secondaryIdentity}</span>
                    ) : null}
                  </div>
                </div>

                {localizedRoles.length > 0 ? (
                  <div className="profile-dropdown__roles">
                    {localizedRoles.map((role) => (
                      <span key={role} className="role-chip role-chip--profile">
                        {role}
                      </span>
                    ))}
                  </div>
                ) : null}

                <div className="profile-dropdown__language">
                  <LanguageSwitcher variant="dropdown" />
                </div>

                <button type="button" className="profile-dropdown__logout" onClick={handleLogout} role="menuitem">
                  <span className="profile-dropdown__logout-icon" aria-hidden="true">
                    <AppIcon name="logout" />
                  </span>
                  <span>{t("common.logout")}</span>
                </button>
              </div>
            ) : null}
          </div>
        </header>

        <main className="content">
          <div className="content-inner">
            <Outlet />
          </div>
        </main>
      </div>
    </div>
  );
}
