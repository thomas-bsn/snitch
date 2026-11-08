package httpapi

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5"
)

// A publisher key is installed by the operator, never by an HTTP request. Importing
// an approved release reads no remote URL and executes no package or model code.
const engineTrustEnv = "SNITCH_ENGINE_TRUST_KEYS"

var engineVersion = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)
var engineCUDA = regexp.MustCompile(`^(cpu|[0-9]{1,2}\.[0-9]{1,2})$`)
var engineDigest = regexp.MustCompile(`^[0-9a-f]{64}$`)

type engineWheel struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}
type engineManifest struct {
	Schema         int           `json:"schema"`
	Engine         string        `json:"engine"`
	Version        string        `json:"version"`
	AdapterVersion int           `json:"adapter_version"`
	Python         string        `json:"python"`
	Platform       string        `json:"platform"`
	CUDA           string        `json:"cuda"`
	License        string        `json:"license"`
	Kind           string        `json:"kind"`
	Architecture   string        `json:"architecture"`
	DiskBytes      int64         `json:"disk_bytes"`
	VRAMBytes      int64         `json:"vram_bytes"`
	Wheels         []engineWheel `json:"wheels"`
}

// This is the protocol implemented in the Worker, not an extensible import path.
func validEngineManifest(m engineManifest) bool {
	architecture := map[string]string{"pyannote": "pyannote", "nemo": "sortformer", "nemo-titanet": "titanet", "speechbrain-ecapa": "ecapa-tdnn"}[m.Engine]
	kindOK := (m.Engine == "pyannote" && (m.Kind == "diarization" || m.Kind == "fingerprint")) || (m.Engine == "nemo" && m.Kind == "diarization") || ((m.Engine == "nemo-titanet" || m.Engine == "speechbrain-ecapa") && m.Kind == "fingerprint")
	if m.Schema != 1 || m.AdapterVersion != 1 || architecture == "" || m.Architecture != architecture || !kindOK || !engineVersion.MatchString(m.Version) || m.Python != "3.12" || m.Platform != "linux-x86_64" || !engineCUDA.MatchString(m.CUDA) || strings.TrimSpace(m.License) == "" || len(m.License) > 100 || m.DiskBytes <= 0 || m.DiskBytes > 40<<30 || (m.VRAMBytes < 0 || m.VRAMBytes > 256<<30) || len(m.Wheels) == 0 || len(m.Wheels) > 256 {
		return false
	}
	names := map[string]bool{}
	var size int64
	for _, w := range m.Wheels {
		u, err := url.Parse(w.URL)
		if err != nil || u.Scheme != "https" || u.Host != "files.pythonhosted.org" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || !strings.HasSuffix(w.Name, ".whl") || strings.ContainsAny(w.Name, "/\\") || strings.Contains(w.Name, "..") || len(w.Name) > 240 || !engineDigest.MatchString(w.SHA256) || w.Size <= 0 || w.Size > 3<<30 || names[w.Name] {
			return false
		}
		names[w.Name] = true
		size += w.Size
	}
	return size <= m.DiskBytes
}

type engineRelease struct {
	Digest   string          `json:"digest"`
	Manifest json.RawMessage `json:"manifest"`
	Revoked  bool            `json:"revoked"`
}

func verifyEngineRelease(keyID, payload, signature string, keys map[string]string) (engineManifest, []byte, string, bool) {
	var m engineManifest
	raw, err := base64.StdEncoding.DecodeString(payload)
	if err != nil || len(raw) > 256<<10 {
		return m, nil, "", false
	}
	key, err := base64.StdEncoding.DecodeString(keys[keyID])
	if err != nil || len(key) != ed25519.PublicKeySize {
		return m, nil, "", false
	}
	sig, err := base64.StdEncoding.DecodeString(signature)
	if err != nil || !ed25519.Verify(key, raw, sig) {
		return m, nil, "", false
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil || len(fields) != 13 {
		return m, nil, "", false
	}
	for _, value := range fields {
		if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return m, nil, "", false
		}
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&m) != nil || decoder.Decode(new(any)) != io.EOF || !validEngineManifest(m) {
		return m, nil, "", false
	}
	sum := sha256.Sum256(raw)
	return m, raw, hex.EncodeToString(sum[:]), true
}

func (s *Server) registerVoiceEngineRoutes(mux *http.ServeMux) {
	mux.Handle("GET /api/admin/voice-engines", s.protected(true, http.HandlerFunc(s.listVoiceEngines)))
	mux.Handle("POST /api/admin/voice-engines", s.protectedFor(isSuperAdministrator, http.HandlerFunc(s.importVoiceEngine)))
	mux.Handle("DELETE /api/admin/voice-engines/{digest}", s.protectedFor(isSuperAdministrator, http.HandlerFunc(s.revokeVoiceEngine)))
}
func (s *Server) listVoiceEngines(w http.ResponseWriter, r *http.Request) {
	rows, err := s.Pool.Query(r.Context(), `SELECT digest,manifest,revoked FROM voice_engine_registry ORDER BY engine,created_at DESC`)
	if err != nil {
		s.internal(w, err)
		return
	}
	defer rows.Close()
	out := []engineRelease{}
	for rows.Next() {
		var v engineRelease
		if err = rows.Scan(&v.Digest, &v.Manifest, &v.Revoked); err != nil {
			s.internal(w, err)
			return
		}
		out = append(out, v)
	}
	if err = rows.Err(); err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]any{"engines": out})
}
func (s *Server) importVoiceEngine(w http.ResponseWriter, r *http.Request) {
	var input struct {
		KeyID     string `json:"key_id"`
		Payload   string `json:"payload"`
		Signature string `json:"signature"`
	}
	if !decode(w, r, &input) {
		return
	}
	keys := map[string]string{}
	if json.Unmarshal([]byte(os.Getenv(engineTrustEnv)), &keys) != nil || len(keys) == 0 {
		fail(w, http.StatusServiceUnavailable, "engine_trust_unconfigured")
		return
	}
	m, raw, digest, ok := verifyEngineRelease(input.KeyID, input.Payload, input.Signature, keys)
	if !ok {
		fail(w, http.StatusBadRequest, "engine_release_untrusted")
		return
	}
	ss := r.Context().Value(sessionKey{}).(session)
	// Versions are immutable, including after revocation. A revoked version cannot be resurrected by replay.
	tag, err := s.Pool.Exec(r.Context(), `INSERT INTO voice_engine_registry(digest,engine,version,kind,key_id,signed_payload,signature,manifest,author_id,author_name) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) ON CONFLICT DO NOTHING`, digest, m.Engine, m.Version, m.Kind, input.KeyID, input.Payload, input.Signature, raw, ss.User.ID, ss.User.Username)
	if err != nil {
		s.internal(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		fail(w, http.StatusConflict, "engine_version_exists")
		return
	}
	s.Log.Info("voice engine approved", "engine", m.Engine, "version", m.Version, "digest", digest, "author_id", ss.User.ID)
	reply(w, http.StatusCreated, engineRelease{Digest: digest, Manifest: raw})
}
func (s *Server) revokeVoiceEngine(w http.ResponseWriter, r *http.Request) {
	digest := r.PathValue("digest")
	if !engineDigest.MatchString(digest) {
		fail(w, http.StatusBadRequest, "invalid_engine_digest")
		return
	}
	// Serialize with model order creation: no order can acquire a revoked release.
	err := s.withVoiceModelLock(r.Context(), func(tx pgx.Tx) error {
		tag, err := tx.Exec(r.Context(), `UPDATE voice_engine_registry SET revoked=true WHERE digest=$1`, digest)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return orderFailure{http.StatusNotFound, "engine_not_found"}
		}
		// Pending deliveries must not install a revoked release. Acknowledged work drains normally;
		// existing active runtimes are never removed by a registry revocation.
		_, err = tx.Exec(r.Context(), `WITH cancelled AS (UPDATE worker_model_orders SET status='cancelled',cause_code='',finished_at=now() WHERE status='pending' AND runtime=(SELECT manifest FROM voice_engine_registry WHERE digest=$1) RETURNING id) UPDATE worker_voice_models SET state='failed',cause_code='runtime_incompatible',updated_at=now() WHERE order_id IN (SELECT id FROM cancelled)`, digest)
		if err != nil {
			return err
		}
		// Queued engine orders of this release are cancelled the same way; their inventory row is not a promise.
		if _, err = tx.Exec(r.Context(), `WITH cancelled AS (UPDATE worker_engine_orders SET status='cancelled',finished_at=now() WHERE status='pending' AND digest=$1 RETURNING id,worker_id,generation) UPDATE worker_engine_inventory i SET state='failed',cause_code='runtime_incompatible',updated_at=now() FROM cancelled c WHERE i.order_id=c.id`, digest); err != nil {
			return err
		}
		_, err = tx.Exec(r.Context(), `UPDATE worker_model_replacements r SET cause_code='runtime_incompatible',updated_at=now() WHERE phase='preparing' AND EXISTS(SELECT 1 FROM worker_voice_models m JOIN worker_model_orders o ON o.id=m.order_id WHERE m.worker_id=r.worker_id AND m.model_id=r.new_model_id AND o.status='cancelled' AND o.runtime=(SELECT manifest FROM voice_engine_registry WHERE digest=$1))`, digest)
		return err
	})
	if err != nil {
		if !s.writeOrderFailure(w, err) {
			s.internal(w, err)
		}
		return
	}
	ss := r.Context().Value(sessionKey{}).(session)
	s.Log.Info("voice engine revoked", "digest", digest, "author_id", ss.User.ID)
	reply(w, http.StatusOK, map[string]bool{"revoked": true})
}

func approvedEngine(ctx context.Context, db pgQuerier, engine, kind string, digest ...string) (json.RawMessage, error) {
	var release json.RawMessage
	if len(digest) > 0 && digest[0] != "" {
		if !engineDigest.MatchString(digest[0]) {
			return nil, orderFailure{http.StatusBadRequest, "invalid_engine_digest"}
		}
		err := db.QueryRow(ctx, `SELECT manifest FROM voice_engine_registry WHERE digest=$1 AND engine=$2 AND manifest->>'kind'=$3 AND NOT revoked`, digest[0], engine, kind).Scan(&release)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, orderFailure{http.StatusConflict, "engine_release_unavailable"}
		}
		return release, err
	}
	err := db.QueryRow(ctx, `SELECT manifest FROM voice_engine_registry WHERE engine=$1 AND NOT revoked AND manifest->>'kind'=$2 ORDER BY created_at DESC,digest LIMIT 1`, engine, kind).Scan(&release)
	if errors.Is(err, pgx.ErrNoRows) {
		// Existing installations remain usable without a registry. New adapters fail closed.
		var registered bool
		if e := db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM voice_engine_registry WHERE engine=$1 AND manifest->>'kind'=$2)`, engine, kind).Scan(&registered); e != nil {
			return nil, e
		}
		if !registered && (engine == "pyannote" || engine == "nemo" || engine == "silero-jit") {
			return nil, nil
		}
		return nil, orderFailure{http.StatusConflict, "engine_not_installed"}
	}
	return release, err
}
