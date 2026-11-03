import { MembersTab } from "./admin/members/MembersTab";
import { SignupsTab } from "./admin/members/SignupsTab";
import { ExportsTab } from "./admin/members/ExportsTab";
import { VoiceTab } from "./admin/voice/VoiceTab";
import { VocabularyTab } from "./admin/transcription/VocabularyTab";
import { TranscriptionTab } from "./admin/transcription/TranscriptionTab";
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
import { FoldersTab } from "./admin/folders/FoldersTab";
import { QueueTab } from "./admin/queue/QueueTab";
import { SemanticTab } from "./admin/semantic/SemanticTab";
import { WorkersTab } from "./admin/workers/WorkersTab";
import { useServerUnreachable } from "./api/connection";
import { I18nProvider } from "./i18n";
import { Forbidden, NotFound } from "./pages/pages";
import { isAdministrator } from "./roles";
import { AccountPage } from "./screens/account/AccountPage";
import { ConnectionLost, Splash } from "./screens/ConnectionLost";
import { ClipPage } from "./library/ClipPage";
import { ImportPage } from "./import/ImportPage";
import { LibraryPage } from "./library/LibraryPage";
import { LoginPage } from "./screens/LoginPage";
import { RegisterPage } from "./screens/RegisterPage";
import { SessionProvider, useSession } from "./session";
import { SetupWizard } from "./setup/SetupWizard";
import { AppShell } from "./shell/AppShell";
import { GroupsTab } from "./people/GroupsTab";
import { PeopleLayout } from "./people/PeopleLayout";
import { PersonList } from "./people/PersonList";
import { PersonPage } from "./people/PersonPage";
import { VerificationTab } from "./people/VerificationTab";
import { SearchPage } from "./search/SearchPage";
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
  if (setup && !setup.initialized) return <SetupWizard />;

  return (
    <Routes>
      <Route index element={<Home />} />
      <Route path="login" element={<LoginPage />} />
      <Route path="register" element={<RegisterPage />} />
      <Route element={<RequireSession />}>
        <Route element={<RequireAdministrator />}>
          <Route path="setup" element={<SetupWizard />} />
        </Route>
        <Route element={<AppShell />}>
          <Route path="library" element={<LibraryPage />} />
          <Route path="clips/:id" element={<ClipPage />} />
          <Route path="search" element={<SearchPage />} />
          <Route path="people" element={<PeopleLayout />}>
            <Route index element={<PersonList />} />
            <Route path="verification" element={<VerificationTab />} />
            <Route path="groups" element={<GroupsTab />} />
            <Route path=":id" element={<PersonPage />} />
            <Route path="*" element={<NotFound />} />
          </Route>
          <Route path="import" element={<ImportPage />} />
          <Route path="account/*" element={<AccountPage />} />
          <Route element={<RequireAdministrator />}>
            <Route path="admin" element={<AdminLayout />}>
              <Route index element={<Navigate to="workers" replace />} />
              <Route path="users" element={<MembersTab />} />
              <Route path="users/signups" element={<SignupsTab />} />
              <Route path="exports" element={<ExportsTab />} />
              <Route path="workers" element={<WorkersTab />} />
              <Route path="queue" element={<QueueTab />} />
              <Route path="folders" element={<FoldersTab />} />
              <Route path="semantic" element={<SemanticTab />} />
              <Route path="workers/:id" element={<WorkersTab />} />
              <Route path="voice" element={<VoiceTab />} />
              <Route path="transcription" element={<TranscriptionTab />} />
              <Route
                path="transcription/vocabulary"
                element={<VocabularyTab />}
              />
              {ADMIN_TABS.filter(
                (tab) =>
                  ![
                    "workers",
                    "queue",
                    "folders",
                    "semantic",
                    "voice",
                    "transcription",
                    "users",
                    "exports",
                  ].includes(tab.path),
              ).map((tab) => (
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
