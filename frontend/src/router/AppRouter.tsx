import { Navigate, Route, Routes } from "react-router-dom";

import { ProtectedRoute } from "../components/ProtectedRoute";
import { RoleGuard } from "../components/RoleGuard";
import { AppLayout } from "../layouts/AppLayout";
import { BookingRulesPage } from "../pages/BookingRulesPage";
import { BookingsPage } from "../pages/BookingsPage";
import { DashboardPage } from "../pages/DashboardPage";
import { ForbiddenPage } from "../pages/ForbiddenPage";
import { LoginPage } from "../pages/LoginPage";
import { MyBookingsPage } from "../pages/MyBookingsPage";
import { ResourceDetailsPage } from "../pages/ResourceDetailsPage";
import { ResourcesPage } from "../pages/ResourcesPage";
import { UsersPage } from "../pages/UsersPage";

export function AppRouter(): JSX.Element {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />

      <Route element={<ProtectedRoute />}>
        <Route element={<AppLayout />}>
          <Route path="/" element={<DashboardPage />} />
          <Route path="/resources" element={<ResourcesPage />} />
          <Route path="/resources/:id" element={<ResourceDetailsPage />} />
          <Route path="/my-bookings" element={<MyBookingsPage />} />
          <Route path="/forbidden" element={<ForbiddenPage />} />

          <Route element={<RoleGuard allowedRoles={["admin", "manager"]} />}>
            <Route path="/bookings" element={<BookingsPage />} />
          </Route>

          <Route element={<RoleGuard allowedRoles={["admin"]} />}>
            <Route path="/users" element={<UsersPage />} />
            <Route path="/booking-rules" element={<BookingRulesPage />} />
          </Route>
        </Route>
      </Route>

      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
