import {
  BrowserRouter,
  Navigate,
  Outlet,
  Route,
  Routes,
  useLocation,
} from "react-router";
import { AdminLayout, AdminPlaceholder } from "./admin/AdminLayout";
import { ADMIN_TABS } from "./admin/tabs";
import { WorkersTab } from "./admin/workers/WorkersTab";
import { useServerUnreachable } from "./api/connection";
import { I18nProvider } from "./i18n";
import {
  AccountPage,
  Forbidden,
  ImportPage,
  NotFound,
  PeoplePage,
  SearchPage,
} from "./pages/pages";
import { isAdministrator } from "./roles";
import { ConnectionLost, Splash } from "./screens/ConnectionLost";
import { ClipPage } from "./library/ClipPage";
import { LibraryPage } from "./library/LibraryPage";
import { InstallationPending } from "./screens/InstallationPending";
import { LoginPage } from "./screens/LoginPage";
import { RegisterPage } from "./screens/RegisterPage";
import { SessionProvider, useSession } from "./session";
import { AppShell } from "./shell/AppShell";
import { ThemeProvider } from "./theme";

/** Screens that need a signed-in person; others are sent to the sign-in screen. */
function RequireSession() {
  const { user } = useSession();
  const location = useLocation();
  if (!user) {
    return (
      <Navigate
        to="/login"
        replace
        state={{ from: location.pathname + location.search }}
      />
    );
  }
  return <Outlet />;
}

/** Administration screens: reserved to administrators. */
function RequireAdministrator() {
  const { user } = useSession();
  if (!user || !isAdministrator(user)) return <Forbidden />;
  return <Outlet />;
}

/** The root address: where the Discord return also lands, with its notice. */
function Home() {
  const { user } = useSession();
  const { search } = useLocation();
  return <Navigate to={`${user ? "/library" : "/login"}${search}`} replace />;
}

function Routing() {
  const { phase, setup, reload } = useSession();
  const unreachable = useServerUnreachable();

  if (unreachable || phase === "failed") {
    return <ConnectionLost onRetry={() => void reload()} />;
  }
  if (phase === "loading") return <Splash />;
  if (setup && !setup.initialized) return <InstallationPending />;

  return (
    <Routes>
      <Route index element={<Home />} />
      <Route path="login" element={<LoginPage />} />
      <Route path="register" element={<RegisterPage />} />
      <Route element={<RequireSession />}>
        <Route element={<AppShell />}>
          <Route path="library" element={<LibraryPage />} />
          <Route path="clips/:id" element={<ClipPage />} />
          <Route path="search" element={<SearchPage />} />
          <Route path="people" element={<PeoplePage />} />
          <Route path="import" element={<ImportPage />} />
          <Route path="account/*" element={<AccountPage />} />
          <Route element={<RequireAdministrator />}>
            <Route path="admin" element={<AdminLayout />}>
              <Route index element={<Navigate to="workers" replace />} />
              <Route path="workers" element={<WorkersTab />} />
              <Route path="workers/:id" element={<WorkersTab />} />
              {ADMIN_TABS.filter((tab) => tab.path !== "workers").map((tab) => (
                <Route
                  key={tab.path}
                  path={`${tab.path}/*`}
                  element={<AdminPlaceholder />}
                />
              ))}
              <Route path="*" element={<NotFound />} />
            </Route>
          </Route>
          <Route path="*" element={<NotFound />} />
        </Route>
      </Route>
      <Route path="*" element={<Home />} />
    </Routes>
  );
}

export function App() {
  return (
    <ThemeProvider>
      <I18nProvider>
        <SessionProvider>
          <BrowserRouter>
            <Routing />
          </BrowserRouter>
        </SessionProvider>
      </I18nProvider>
    </ThemeProvider>
  );
}
