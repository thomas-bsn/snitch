package workers

import (
	"errors"
	"regexp"
	"strings"
	"text/template"
)

// Placeholder written for every value the administrator must provide by hand.
const Placeholder = "A_RENSEIGNER"

// Bundle is the ready-to-run Docker Compose project handed to an administrator when a
// worker is created. It is generated once, in memory, from the new token: nothing is stored.
type Bundle struct {
	Compose string `json:"compose"`
	Env     string `json:"env"`
	// InsecureAPIURL is true when the public URL is not HTTPS: the worker refuses it.
	InsecureAPIURL bool `json:"insecure_api_url"`
}

type BundleInput struct {
	APIURL, Token, Image string
	// WhisperURL is the address of this worker's Whisper server, when the administrator gave it.
	WhisperURL string
}

// composeFile is static on purpose: every variable part lives in the .env file, so no value
// is ever interpolated into YAML. It mirrors deploy/worker/compose.yaml.
const composeFile = `# Worker Snitch avec GPU NVIDIA, généré par Snitch pour ce worker.
# Placer ce fichier et le fichier .env dans un même dossier du PC, renseigner les valeurs
# « A_RENSEIGNER » du .env, puis lancer : docker compose up -d
# Le worker ne publie aucun port : il appelle l'API Snitch en HTTPS sortant uniquement.
services:
  worker:
    image: ${SNITCH_WORKER_IMAGE:?Renseigner SNITCH_WORKER_IMAGE dans .env}
    pull_policy: always
    restart: unless-stopped
    environment:
      SNITCH_API_URL: ${SNITCH_API_URL:?Renseigner SNITCH_API_URL (HTTPS) dans .env}
      SNITCH_WORKER_TOKEN: ${SNITCH_WORKER_TOKEN:?Renseigner SNITCH_WORKER_TOKEN dans .env}
      SNITCH_WORKER_CA_BUNDLE: ${SNITCH_WORKER_CA_BUNDLE:-}
      SNITCH_WORKER_CAPACITY: ${SNITCH_WORKER_CAPACITY:-1}
      SNITCH_WORKER_MACHINE_NAME: ${SNITCH_WORKER_MACHINE_NAME:-}
      TRANSCRIPTION_API_URL: ${TRANSCRIPTION_API_URL:?Renseigner TRANSCRIPTION_API_URL dans .env}
      TRANSCRIPTION_MODEL: ${TRANSCRIPTION_MODEL:-}
      TRANSCRIPTION_API_KEY: ${TRANSCRIPTION_API_KEY:-}
      SNITCH_VOICE_DIARIZATION_MODEL_PATH: ${SNITCH_VOICE_DIARIZATION_MODEL_PATH:-/models/diarization}
      SNITCH_VOICE_EMBEDDING_MODEL_PATH: ${SNITCH_VOICE_EMBEDDING_MODEL_PATH:-/models/embedding}
      SNITCH_VOICE_EMBEDDING_MODEL_VERSION: ${SNITCH_VOICE_EMBEDDING_MODEL_VERSION:-}
      SNITCH_VOICE_EMBEDDING_MODEL_NAME: ${SNITCH_VOICE_EMBEDDING_MODEL_NAME:-pyannote/wespeaker-voxceleb-resnet34-LM}
      SNITCH_VOICE_DEVICE: ${SNITCH_VOICE_DEVICE:-cuda}
      # Téléchargement des modèles au premier démarrage (HF_TOKEN ne sert qu'à ce téléchargement).
      SNITCH_VOICE_DOWNLOAD_ON_START: ${SNITCH_VOICE_DOWNLOAD_ON_START:-1}
      SNITCH_VOICE_CONFIRM_TERMS: ${SNITCH_VOICE_CONFIRM_TERMS:-1}
      HF_TOKEN: ${HF_TOKEN:-}
    volumes:
      # Volume nommé : les modèles téléchargés survivent aux mises à jour de l'image.
      - snitch-models:/models
    # Joindre un Whisper ou un Ollama qui tourne directement sur le PC (hors Docker) :
    # TRANSCRIPTION_API_URL=http://host.docker.internal:8000
    extra_hosts:
      - "host.docker.internal:host-gateway"
    # Accès GPU (NVIDIA Container Toolkit requis sur l'hôte).
    deploy:
      resources:
        reservations:
          devices:
            - driver: nvidia
              count: 1
              capabilities: [gpu]
volumes:
  snitch-models:
`

var envFile = template.Must(template.New("env").Funcs(template.FuncMap{"v": envValue}).Parse(`# Fichier .env généré par Snitch pour ce worker, à placer à côté de compose.yaml (nom .env, ou
# docker compose --env-file <fichier> up -d).
# Il contient le jeton du worker : ne jamais le versionner ni le partager.
{{if .Insecure}}# ATTENTION : l'URL publique de Snitch n'est pas en HTTPS et le worker refuse le HTTP.
# Passer l'instance en HTTPS (PUBLIC_URL) ou corriger SNITCH_API_URL ci-dessous.
{{end}}
# --- Connexion à Snitch (déjà renseignée) ---
SNITCH_API_URL={{v .APIURL}}
SNITCH_WORKER_TOKEN={{v .Token}}
SNITCH_WORKER_IMAGE={{v .Image}}

# --- Transcription ---
# Serveur compatible POST /v1/audio/transcriptions, par exemple un Whisper lancé sur le PC :
# http://host.docker.internal:8000
TRANSCRIPTION_API_URL={{v .Whisper}}
# Facultatif : le nom que CE serveur donne au modèle de la bibliothèque, seulement s'il diffère.
# Par défaut, le worker utilise le modèle de la bibliothèque, réglé dans Snitch.
# TRANSCRIPTION_MODEL=
# Clé bearer, facultative.
# TRANSCRIPTION_API_KEY=

# --- Modèles vocaux ---
# Jeton Hugging Face, à renseigner : il sert uniquement au premier téléchargement des modèles
# et n'est jamais lu par le worker.
HF_TOKEN={{v .Placeholder}}
SNITCH_VOICE_DOWNLOAD_ON_START=1
SNITCH_VOICE_CONFIRM_TERMS=1
# Révision des modèles, si le démarrage ne la renseigne pas lui-même.
# SNITCH_VOICE_EMBEDDING_MODEL_VERSION=

# --- Facultatif ---
# SNITCH_WORKER_CAPACITY=1
# SNITCH_WORKER_MACHINE_NAME=PC-jeu
# SNITCH_WORKER_CA_BUNDLE=
`))

var safeEnvValue = regexp.MustCompile(`^[A-Za-z0-9_./:@%+=,-]+$`)

// envValue renders a Docker Compose .env value: bare when it only holds safe characters,
// otherwise double-quoted with backslash, quote, dollar sign and newlines escaped.
func envValue(value string) string {
	if safeEnvValue.MatchString(value) {
		return value
	}
	replacer := strings.NewReplacer(`\`, `\\`, `"`, `\"`, `$`, `\$`, "\n", `\n`, "\r", `\r`)
	return `"` + replacer.Replace(value) + `"`
}

func NewBundle(input BundleInput) (Bundle, error) {
	if input.APIURL == "" || input.Token == "" || input.Image == "" {
		return Bundle{}, errors.New("worker bundle needs an API URL, a token and an image")
	}
	insecure := !strings.HasPrefix(strings.ToLower(input.APIURL), "https://")
	whisper := input.WhisperURL
	if whisper == "" {
		whisper = Placeholder
	}
	var env strings.Builder
	data := struct {
		APIURL, Token, Image, Placeholder, Whisper string
		Insecure                                   bool
	}{input.APIURL, input.Token, input.Image, Placeholder, whisper, insecure}
	if err := envFile.Execute(&env, data); err != nil {
		return Bundle{}, err
	}
	return Bundle{Compose: composeFile, Env: env.String(), InsecureAPIURL: insecure}, nil
}
