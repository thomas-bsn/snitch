// PROTOTYPE — Administration › Users & access › Members: change a role, delete a user (optionally hiding their clips). Stubbed, no writes.
import { useEffect, useState } from "react";

type Role = "admin" | "user";
type U = {
  id: number;
  name: string;
  role: Role;
  joined: string;
  clips: number;
  me?: boolean;
};
const SEED: U[] = [
  {
    id: 1,
    name: "Thomas",
    role: "admin",
    joined: "Sept 2",
    clips: 130,
    me: true,
  },
  { id: 2, name: "Léa", role: "user", joined: "Sept 9", clips: 92 },
  { id: 3, name: "Marc", role: "user", joined: "Sept 12", clips: 61 },
  { id: 4, name: "Alex", role: "user", joined: "Oct 1", clips: 4 },
];

export function Members() {
  const [users, setUsers] = useState(SEED);
  const [roleFor, setRoleFor] = useState<{ u: U; to: Role } | null>(null);
  const [del, setDel] = useState<U | null>(null);
  const [hide, setHide] = useState(false);
  const [toast, setToast] = useState("");
  const admins = users.filter((u) => u.role === "admin").length;

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        setRoleFor(null);
        setDel(null);
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);

  const say = (m: string) => {
    setToast(m);
    setTimeout(() => setToast(""), 4000);
  };
  const lockedReason = (u: U) =>
    u.me
      ? "That is you. Use Account settings."
      : u.role === "admin" && admins === 1
        ? "The last administrator cannot be changed."
        : "";

  return (
    <div className="verif">
      <div className="panel">
        <table>
          <thead>
            <tr>
              <th>Member</th>
              <th>Role</th>
              <th>Joined</th>
              <th>Clips</th>
              <th />
            </tr>
          </thead>
          <tbody>
            {users.map((u) => {
              const locked = lockedReason(u);
              return (
                <tr key={u.id}>
                  <td>
                    <b>{u.name}</b>{" "}
                    {u.me && <span className="pill info">You</span>}
                  </td>
                  <td>
                    <select
                      className="rolesel"
                      aria-label={`Role of ${u.name}`}
                      value={u.role}
                      disabled={!!locked}
                      title={locked}
                      onChange={(e) =>
                        setRoleFor({ u, to: e.target.value as Role })
                      }
                    >
                      <option value="user">User</option>
                      <option value="admin">Administrator</option>
                    </select>
                  </td>
                  <td>{u.joined}</td>
                  <td>{u.clips}</td>
                  <td style={{ textAlign: "right" }}>
                    <button
                      className="btn bad"
                      disabled={!!locked}
                      title={locked}
                      onClick={() => {
                        setDel(u);
                        setHide(false);
                      }}
                    >
                      Delete…
                    </button>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
      <span className="mute">
        The last administrator and your own account cannot be changed or deleted
        here.
      </span>

      {roleFor && (
        <>
          <div className="scrim dim" onClick={() => setRoleFor(null)} />
          <div
            className="modal"
            style={{ width: "min(460px,92vw)" }}
            role="alertdialog"
            aria-modal="true"
            aria-labelledby="rl-t"
          >
            <h2 id="rl-t" style={{ margin: 0 }}>
              {roleFor.to === "admin"
                ? `Make ${roleFor.u.name} an administrator?`
                : `Make ${roleFor.u.name} a regular user?`}
            </h2>
            <p className="mute" style={{ margin: 0 }}>
              {roleFor.to === "admin"
                ? "Administrators manage access, workers, folders and settings. They can also change other roles."
                : "They lose access to Administration. Their clips and corrections stay."}
            </p>
            <div className="row" style={{ justifyContent: "flex-end" }}>
              <button className="btn" onClick={() => setRoleFor(null)}>
                Cancel
              </button>
              <button
                className="btn pri"
                onClick={() => {
                  setUsers(
                    users.map((x) =>
                      x.id === roleFor.u.id ? { ...x, role: roleFor.to } : x,
                    ),
                  );
                  say(
                    `${roleFor.u.name} is now ${roleFor.to === "admin" ? "an administrator" : "a regular user"}.`,
                  );
                  setRoleFor(null);
                }}
              >
                Confirm
              </button>
            </div>
          </div>
        </>
      )}

      {del && (
        <>
          <div className="scrim dim" onClick={() => setDel(null)} />
          <div
            className="modal"
            style={{ width: "min(480px,92vw)" }}
            role="alertdialog"
            aria-modal="true"
            aria-labelledby="dl-t"
          >
            <h2 id="dl-t" style={{ margin: 0 }}>
              Delete {del.name}?
            </h2>
            <p className="mute" style={{ margin: 0 }}>
              {del.name} will no longer be able to sign in. Their {del.clips}{" "}
              clips are not deleted.
            </p>
            <label className="check">
              <input
                type="checkbox"
                checked={hide}
                onChange={(e) => setHide(e.target.checked)}
              />
              <span>
                <b>Also hide their clips from the library.</b>
                <br />
                <span className="mute">
                  Otherwise they stay visible to everyone who could see them,
                  shown as “deleted user”.
                </span>
              </span>
            </label>
            <div className="row" style={{ justifyContent: "flex-end" }}>
              <button className="btn" onClick={() => setDel(null)}>
                Cancel
              </button>
              <button
                className="btn bad solid"
                onClick={() => {
                  setUsers(users.filter((x) => x.id !== del.id));
                  say(
                    `${del.name} was deleted. ${hide ? `${del.clips} clips are hidden.` : `${del.clips} clips stay visible.`}`,
                  );
                  setDel(null);
                }}
              >
                Delete {del.name}
              </button>
            </div>
          </div>
        </>
      )}
      {toast && (
        <div className="toast" role="status">
          {toast}
        </div>
      )}
    </div>
  );
}
