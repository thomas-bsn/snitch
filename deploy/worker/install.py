#!/usr/bin/env python3
"""
Snitch Worker — Single-Command Installer

Installs the Snitch Worker on a gaming PC (Windows with WSL2 + Docker Desktop, or Linux).
Usage:
    python install.py                    # Interactive install
    python install.py --diagnose         # Check prerequisites and common errors
    python install.py --update           # Update existing installation (preserves config)
    python install.py --help             # Show all options
"""

from __future__ import annotations

import argparse
import json
import os
import platform
import shutil
import subprocess
import sys
import textwrap
from pathlib import Path
from typing import Optional

# --- Constants ---

REPO_ROOT = Path(__file__).parent.parent.parent.resolve()
WORKER_DIR = REPO_ROOT / "deploy" / "worker"
COMPOSE_FILE = WORKER_DIR / "compose.yaml"
ENV_EXAMPLE = WORKER_DIR / ".env.example"
ENV_FILE = WORKER_DIR / ".env"
MODELS_DIR_DEFAULT = Path.home() / "snitch-models"

REQUIRED_DOCKER_VERSION = (20, 10)
REQUIRED_COMPOSE_VERSION = (2, 0)

SUPPORTED_PLATFORMS = {
    "linux": "Linux (native Docker)",
    "windows": "Windows 10/11 with WSL2 + Docker Desktop",
    "darwin": "macOS (Docker Desktop) — for testing only, not a production target",
}

# --- Colors for output ---

class Colors:
    if sys.stdout.isatty():
        GREEN = "\033[92m"
        YELLOW = "\033[93m"
        RED = "\033[91m"
        BLUE = "\033[94m"
        CYAN = "\033[96m"
        BOLD = "\033[1m"
        RESET = "\033[0m"
    else:
        GREEN = YELLOW = RED = BLUE = CYAN = BOLD = RESET = ""

def info(msg: str) -> None:
    print(f"{Colors.BLUE}ℹ{Colors.RESET}  {msg}")

def success(msg: str) -> None:
    print(f"{Colors.GREEN}✓{Colors.RESET}  {msg}")

def warn(msg: str) -> None:
    print(f"{Colors.YELLOW}⚠{Colors.RESET}  {msg}")

def error(msg: str) -> None:
    print(f"{Colors.RED}✗{Colors.RESET}  {msg}", file=sys.stderr)

def bold(msg: str) -> str:
    return f"{Colors.BOLD}{msg}{Colors.RESET}"

# --- Platform detection ---

def detect_platform() -> tuple[str, dict]:
    """Detect the platform and return (platform_key, details)."""
    system = platform.system().lower()
    details = {
        "system": system,
        "release": platform.release(),
        "version": platform.version(),
        "machine": platform.machine(),
        "is_wsl2": False,
        "wsl_version": None,
    }

    if system == "linux":
        # Check for WSL2
        try:
            with open("/proc/version", "r") as f:
                version_info = f.read().lower()
                if "microsoft" in version_info or "wsl" in version_info:
                    details["is_wsl2"] = True
                    # Try to get WSL version
                    result = subprocess.run(
                        ["wsl.exe", "-l", "-v"],
                        capture_output=True,
                        text=True,
                        timeout=5,
                    )
                    if result.returncode == 0:
                        details["wsl_version"] = result.stdout.strip()
        except Exception:
            pass
        return "linux", details

    if system == "windows":
        return "windows", details

    if system == "darwin":
        return "darwin", details

    return "unknown", details

# --- Prerequisite checks ---

def run_command(cmd: list[str], capture: bool = True, timeout: int = 30, cwd: Optional[Path] = None) -> tuple[int, str, str]:
    """Run a command and return (returncode, stdout, stderr)."""
    try:
        result = subprocess.run(
            cmd,
            capture_output=capture,
            text=True,
            timeout=timeout,
            cwd=cwd,
        )
        return result.returncode, result.stdout.strip(), result.stderr.strip()
    except subprocess.TimeoutExpired:
        return -1, "", "timeout"
    except FileNotFoundError:
        return -1, "", "command not found"
    except Exception as e:
        return -1, "", str(e)

def parse_version(version_str: str) -> tuple[int, ...]:
    """Parse a version string like '20.10.17' into a tuple of ints."""
    parts = []
    for part in version_str.split("."):
        try:
            parts.append(int(part))
        except ValueError:
            # Handle cases like "20.10.17-ubuntu"
            digits = ""
            for ch in part:
                if ch.isdigit():
                    digits += ch
                else:
                    break
            if digits:
                parts.append(int(digits))
            else:
                parts.append(0)
    return tuple(parts)

def check_docker() -> tuple[bool, str]:
    """Check if Docker is installed and meets version requirements."""
    code, out, err = run_command(["docker", "version", "--format", "{{.Server.Version}}"])
    if code != 0:
        return False, f"Docker not found or not running: {err or out}"
    version = parse_version(out)
    if version < REQUIRED_DOCKER_VERSION:
        return False, f"Docker version {out} is too old (need >= {'.'.join(map(str, REQUIRED_DOCKER_VERSION))})"
    return True, f"Docker {out} ✓"

def check_docker_compose() -> tuple[bool, str]:
    """Check if Docker Compose v2 is available."""
    code, out, err = run_command(["docker", "compose", "version", "--short"])
    if code != 0:
        return False, f"Docker Compose v2 not found: {err or out}"
    version = parse_version(out.lstrip("v"))
    if version < REQUIRED_COMPOSE_VERSION:
        return False, f"Docker Compose version {out} is too old (need >= {'.'.join(map(str, REQUIRED_COMPOSE_VERSION))})"
    return True, f"Docker Compose {out} ✓"

def check_nvidia_toolkit() -> tuple[bool, str]:
    """Check if NVIDIA Container Toolkit is configured."""
    # Check if nvidia-smi works on host
    code, out, err = run_command(["nvidia-smi"])
    if code != 0:
        return False, "nvidia-smi not found — NVIDIA driver not installed or not in PATH"

    # Check if Docker can access GPU
    code, out, err = run_command(
        ["docker", "run", "--rm", "--gpus", "all", "ubuntu", "nvidia-smi"],
        timeout=60,
    )
    if code != 0:
        return False, f"NVIDIA Container Toolkit not configured for Docker: {err or out}"

    return True, "NVIDIA Container Toolkit ✓"

def check_prerequisites(platform_key: str, details: dict) -> tuple[bool, list[str]]:
    """Check all prerequisites and return (all_ok, messages)."""
    messages = []
    all_ok = True

    info("Checking prerequisites...")

    # Docker
    ok, msg = check_docker()
    messages.append(("ok" if ok else "fail", msg))
    if not ok:
        all_ok = False

    # Docker Compose
    ok, msg = check_docker_compose()
    messages.append(("ok" if ok else "fail", msg))
    if not ok:
        all_ok = False

    # NVIDIA (only required for voice features, but recommended)
    if platform_key in ("linux", "windows"):
        ok, msg = check_nvidia_toolkit()
        messages.append(("ok" if ok else "warn", msg))
        if not ok:
            warn("GPU acceleration unavailable — voice features will be very slow or fail")
    elif platform_key == "darwin":
        messages.append(("warn", "macOS detected — GPU acceleration not available (Apple Silicon uses different GPU stack)"))

    # Platform-specific checks
    if platform_key == "windows":
        if not details["is_wsl2"]:
            messages.append(("warn", "Not running inside WSL2 — install WSL2 and run this script from a WSL2 terminal"))
            all_ok = False
        else:
            messages.append(("ok", "WSL2 detected ✓"))

    return all_ok, messages

# --- Configuration ---

def load_env_example() -> dict[str, str]:
    """Load the .env.example file and return a dict of key -> default value."""
    config = {}
    if not ENV_EXAMPLE.exists():
        error(f".env.example not found at {ENV_EXAMPLE}")
        sys.exit(1)

    with open(ENV_EXAMPLE, "r") as f:
        for line in f:
            line = line.strip()
            if not line or line.startswith("#"):
                continue
            if "=" in line:
                key, value = line.split("=", 1)
                config[key.strip()] = value.strip()
    return config

def prompt_for_config(config: dict[str, str], existing_env: Optional[dict[str, str]] = None) -> dict[str, str]:
    """Interactively prompt for configuration values."""
    print()
    print(bold("=== Configuration ==="))
    print("Press Enter to keep the default value (shown in brackets).")
    print("For secrets, the input will be hidden.")
    print()

    result = dict(config)

    # Required fields
    required_fields = [
        ("SNITCH_API_URL", "HTTPS URL of your Snitch instance (e.g., https://snitch.example.org)", True),
        ("SNITCH_WORKER_TOKEN", "Worker token (shown once when creating the worker in Snitch admin)", True),
        ("TRANSCRIPTION_API_URL", "Whisper API URL (e.g., http://host.docker.internal:8000)", True),
        ("TRANSCRIPTION_MODEL", "Whisper model name recognized by the transcription server", True),
        ("SNITCH_MODELS_DIR", f"Host directory for voice models (default: {MODELS_DIR_DEFAULT})", False),
    ]

    for key, description, is_secret in required_fields:
        default = result.get(key, "")
        # Use existing value if available
        if existing_env and key in existing_env and existing_env[key]:
            default = existing_env[key]

        prompt = f"{key}"
        if default:
            prompt += f" [{default}]"
        prompt += f": {description}"
        print(f"\n{Colors.CYAN}{prompt}{Colors.RESET}")

        if is_secret:
            import getpass
            value = getpass.getpass("> ").strip()
        else:
            value = input("> ").strip()

        if value:
            result[key] = value
        elif default and key not in result:
            result[key] = default

    # Optional fields
    print()
    print(bold("--- Optional (press Enter to skip) ---"))

    optional_fields = [
        ("SNITCH_WORKER_CA_BUNDLE", "Custom CA bundle path in container (for self-signed certs)"),
        ("SNITCH_WORKER_CAPACITY", "Concurrent tasks (1-4, default: 1)"),
        ("SNITCH_WORKER_MACHINE_NAME", "Custom machine name (default: container hostname)"),
        ("TRANSCRIPTION_API_KEY", "Bearer token for transcription API (if required)"),
        ("SNITCH_VOICE_EMBEDDING_MODEL_NAME", "Embedding model name (default: pyannote/wespeaker-voxceleb-resnet34-LM)"),
        ("SNITCH_VOICE_DEVICE", "Compute device: cuda, auto, or cpu (default: cuda)"),
        ("SNITCH_VOICE_MODEL_REVISION", "Pin model revision for reproducibility"),
    ]

    for key, description in optional_fields:
        default = result.get(key, "")
        if existing_env and key in existing_env and existing_env[key]:
            default = existing_env[key]

        prompt = f"{key}"
        if default:
            prompt += f" [{default}]"
        prompt += f": {description}"
        print(f"\n{Colors.CYAN}{prompt}{Colors.RESET}")

        value = input("> ").strip()
        if value:
            result[key] = value

    # Voice model download opt-in
    print()
    print(bold("--- Voice Model Download (opt-in) ---"))
    print("Enable automatic download of pyannote models on first start.")
    print("Requires: HF_TOKEN (read-only Hugging Face token), accepting model terms on HF.")
    print("The token is ONLY used during bootstrap, then removed from worker env.")

    download_on_start = input("\nEnable automatic model download on first start? [y/N]: ").strip().lower()
    if download_on_start in ("y", "yes", "1", "true"):
        result["SNITCH_VOICE_DOWNLOAD_ON_START"] = "1"
        confirm_terms = input("I have accepted the model terms on Hugging Face (pyannote/speaker-diarization-community-1)? [y/N]: ").strip().lower()
        if confirm_terms in ("y", "yes", "1", "true"):
            result["SNITCH_VOICE_CONFIRM_TERMS"] = "1"
        else:
            warn("Terms not confirmed — automatic download will not work")
            result["SNITCH_VOICE_DOWNLOAD_ON_START"] = "0"

        hf_token = getpass.getpass("HF_TOKEN (read-only, will be hidden): ").strip()
        if hf_token:
            result["HF_TOKEN"] = hf_token
        else:
            warn("No HF_TOKEN provided — automatic download will fail")
            result["SNITCH_VOICE_DOWNLOAD_ON_START"] = "0"
    else:
        result["SNITCH_VOICE_DOWNLOAD_ON_START"] = "0"
        result["SNITCH_VOICE_CONFIRM_TERMS"] = "0"

    return result

def load_existing_env() -> Optional[dict[str, str]]:
    """Load existing .env file if present."""
    if not ENV_FILE.exists():
        return None
    config = {}
    with open(ENV_FILE, "r") as f:
        for line in f:
            line = line.strip()
            if not line or line.startswith("#"):
                continue
            if "=" in line:
                key, value = line.split("=", 1)
                config[key.strip()] = value.strip()
    return config

def write_env_file(config: dict[str, str]) -> None:
    """Write the .env file with comments preserved from .env.example."""
    # Read the example to preserve comments and structure
    with open(ENV_EXAMPLE, "r") as f:
        lines = f.readlines()

    output_lines = []
    for line in lines:
        stripped = line.strip()
        if not stripped or stripped.startswith("#"):
            output_lines.append(line)
            continue
        if "=" in stripped:
            key = stripped.split("=", 1)[0].strip()
            if key in config:
                output_lines.append(f"{key}={config[key]}\n")
            else:
                output_lines.append(line)
        else:
            output_lines.append(line)

    # Add any new keys not in example
    example_keys = set()
    for line in lines:
        stripped = line.strip()
        if "=" in stripped and not stripped.startswith("#"):
            example_keys.add(stripped.split("=", 1)[0].strip())

    for key, value in config.items():
        if key not in example_keys:
            output_lines.append(f"{key}={value}\n")

    with open(ENV_FILE, "w") as f:
        f.writelines(output_lines)

    # Secure the file (owner read/write only)
    os.chmod(ENV_FILE, 0o600)
    success(f"Configuration written to {ENV_FILE} (permissions 600)")

# --- Models directory ---

def setup_models_dir(models_dir: str, platform_key: str, details: dict) -> bool:
    """Create and set up the models directory with correct permissions."""
    path = Path(models_dir).expanduser().resolve()

    info(f"Setting up models directory: {path}")
    path.mkdir(parents=True, exist_ok=True)

    if platform_key == "linux" and not details["is_wsl2"]:
        # On native Linux, chown to container user (10001:10001)
        try:
            import pwd
            import grp
            # Check if we can chown (need root)
            if os.geteuid() == 0:
                shutil.chown(path, user="10001", group="10001")
                success(f"Models directory owned by 10001:10001")
            else:
                # Try with sudo
                code, out, err = run_command(["sudo", "chown", "10001:10001", str(path)])
                if code == 0:
                    success(f"Models directory owned by 10001:10001 (via sudo)")
                else:
                    warn(f"Could not chown to 10001:10001 (run manually: sudo chown 10001:10001 {path})")
        except Exception as e:
            warn(f"Could not set ownership: {e}")
    elif platform_key == "windows" or details["is_wsl2"]:
        # On WSL2, avoid /mnt/c paths for models dir if possible
        if str(path).startswith("/mnt/"):
            warn(f"Models directory on Windows drive ({path}) may have permission issues")
            warn("Recommended: use a path under /home/... instead")

    return True

# --- Model download ---

def download_models(models_dir: str, hf_token: str, confirm_terms: bool) -> bool:
    """Download voice models using the download_models.py script."""
    if not confirm_terms:
        error("Model download requires --confirm-terms (accept terms on Hugging Face)")
        return False
    if not hf_token:
        error("Model download requires HF_TOKEN")
        return False

    info("Downloading voice models (this may take several minutes)...")

    diarization_dir = Path(models_dir) / "diarization"
    embedding_dir = Path(models_dir) / "embedding"

    env = os.environ.copy()
    env["HF_TOKEN"] = hf_token

    script = REPO_ROOT / "worker" / "scripts" / "download_models.py"
    code, out, err = run_command([
        sys.executable, str(script),
        "--confirm-terms",
        "--diarization-dir", str(diarization_dir),
        "--embedding-dir", str(embedding_dir),
    ], capture=True, timeout=600, env=env)

    if code != 0:
        error(f"Model download failed: {err or out}")
        return False

    print(out)
    success("Voice models downloaded successfully")
    return True

# --- Docker Compose operations ---

def docker_compose_up(detach: bool = True) -> bool:
    """Run docker compose up in the worker directory."""
    info("Starting worker with docker compose...")
    cmd = ["docker", "compose", "up"]
    if detach:
        cmd.append("-d")
    code, out, err = run_command(cmd, capture=True, timeout=120, cwd=WORKER_DIR)
    if code != 0:
        error(f"docker compose up failed: {err or out}")
        return False
    success("Worker started")
    return True

def docker_compose_pull() -> bool:
    """Pull the latest worker image."""
    info("Pulling latest worker image...")
    code, out, err = run_command(["docker", "compose", "pull"], capture=True, timeout=120, cwd=WORKER_DIR)
    if code != 0:
        error(f"docker compose pull failed: {err or out}")
        return False
    success("Image pulled")
    return True

def docker_compose_logs(follow: bool = True) -> None:
    """Show worker logs."""
    cmd = ["docker", "compose", "logs"]
    if follow:
        cmd.append("-f")
    else:
        cmd.append("--tail=50")
    run_command(cmd, capture=False, cwd=WORKER_DIR)

# --- Diagnostic ---

def run_diagnostics(platform_key: str, details: dict) -> None:
    """Run diagnostic checks and print results."""
    print(bold("=== Snitch Worker Diagnostics ==="))
    platform_name = SUPPORTED_PLATFORMS.get(platform_key, f"Unknown ({platform_key})")
    print(f"Platform: {platform_name} ({details['system']} {details['release']})")
    if details["is_wsl2"]:
        print(f"WSL2: Yes")
    print()

    print(bold("--- Prerequisites ---"))
    all_ok, messages = check_prerequisites(platform_key, details)
    for status, msg in messages:
        if status == "ok":
            print(f"  {Colors.GREEN}✓{Colors.RESET} {msg}")
        elif status == "warn":
            print(f"  {Colors.YELLOW}⚠{Colors.RESET} {msg}")
        else:
            print(f"  {Colors.RED}✗{Colors.RESET} {msg}")

    print()
    print(bold("--- Configuration ---"))
    if ENV_FILE.exists():
        existing = load_existing_env()
        print(f"  {Colors.GREEN}✓{Colors.RESET} .env file exists at {ENV_FILE}")
        # Check required fields
        required = ["SNITCH_API_URL", "SNITCH_WORKER_TOKEN", "TRANSCRIPTION_API_URL", "TRANSCRIPTION_MODEL", "SNITCH_MODELS_DIR"]
        for key in required:
            if existing and existing.get(key):
                print(f"  {Colors.GREEN}✓{Colors.RESET} {key} is set")
            else:
                print(f"  {Colors.RED}✗{Colors.RESET} {key} is missing or empty")
    else:
        print(f"  {Colors.RED}✗{Colors.RESET} No .env file found at {ENV_FILE}")

    print()
    print(bold("--- Models Directory ---"))
    if ENV_FILE.exists():
        existing = load_existing_env()
        models_dir = existing.get("SNITCH_MODELS_DIR", "")
        if models_dir:
            path = Path(models_dir).expanduser().resolve()
            if path.exists():
                diarization = path / "diarization"
                embedding = path / "embedding"
                manifest = path / ".snitch-voice-models.json"
                print(f"  {Colors.GREEN}✓{Colors.RESET} Models directory exists: {path}")
                print(f"     diarization: {'✓' if diarization.exists() else '✗'} ({diarization})")
                print(f"     embedding:   {'✓' if embedding.exists() else '✗'} ({embedding})")
                print(f"     manifest:    {'✓' if manifest.exists() else '✗'} ({manifest})")
                if manifest.exists():
                    try:
                        with open(manifest, "r") as f:
                            data = json.load(f)
                        print(f"     versions: diarization={data.get('diarization', {}).get('revision', '?')[:12]}, embedding={data.get('embedding', {}).get('revision', '?')[:12]}")
                    except Exception:
                        pass
            else:
                print(f"  {Colors.RED}✗{Colors.RESET} Models directory does not exist: {path}")
        else:
            print(f"  {Colors.YELLOW}⚠{Colors.RESET} SNITCH_MODELS_DIR not set in .env")
    else:
        print(f"  {Colors.YELLOW}⚠{Colors.RESET} No .env file to check models directory")

    print()
    print(bold("--- Worker Status ---"))
    code, out, err = run_command(["docker", "compose", "ps", "--format", "json"], capture=True, cwd=WORKER_DIR)
    if code == 0 and out:
        try:
            for line in out.strip().split("\n"):
                if line:
                    container = json.loads(line)
                    print(f"  {Colors.GREEN}✓{Colors.RESET} {container.get('Service', 'worker')}: {container.get('State', 'unknown')} ({container.get('Status', 'unknown')})")
        except Exception:
            print(f"  {Colors.YELLOW}⚠{Colors.RESET} Could not parse docker compose ps output")
    else:
        print(f"  {Colors.YELLOW}⚠{Colors.RESET} No worker containers found (run 'docker compose up -d' to start)")

    print()
    print(bold("--- Common Issues & Fixes ---"))
    issues = [
        ("Worker token not shown", "Create a worker in Snitch Admin → Workers → New Worker. Token is shown ONCE — save it immediately."),
        ("provider_unavailable errors", "Check: models downloaded, SNITCH_VOICE_EMBEDDING_MODEL_VERSION matches manifest, GPU accessible."),
        ("models directory not writable", "On Linux: sudo chown 10001:10001 /path/to/models. On WSL2: use /home/... path, not /mnt/c/..."),
        ("HF_TOKEN not working", "Token must be read-only, and model terms accepted on Hugging Face for the token's account."),
        ("Docker GPU not working", "Install NVIDIA Container Toolkit: https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/latest/install-guide.html"),
        ("WSL2 permission issues", "Use a models directory under /home/, not /mnt/c/. Run: sudo chown 10001:10001 /home/user/snitch-models"),
    ]
    for title, fix in issues:
        print(f"  {Colors.CYAN}{title}:{Colors.RESET} {fix}")

# --- Main commands ---

def cmd_install(args: argparse.Namespace) -> int:
    """Run the interactive installation."""
    print(bold("=== Snitch Worker Installer ==="))
    print()

    # Detect platform
    platform_key, details = detect_platform()
    if platform_key not in SUPPORTED_PLATFORMS:
        error(f"Unsupported platform: {platform_key}")
        error(f"Supported: {', '.join(SUPPORTED_PLATFORMS.values())}")
        return 1

    info(f"Detected platform: {SUPPORTED_PLATFORMS[platform_key]}")

    # Check prerequisites
    all_ok, messages = check_prerequisites(platform_key, details)
    for status, msg in messages:
        if status == "ok":
            print(f"  {Colors.GREEN}✓{Colors.RESET} {msg}")
        elif status == "warn":
            print(f"  {Colors.YELLOW}⚠{Colors.RESET} {msg}")
        else:
            print(f"  {Colors.RED}✗{Colors.RESET} {msg}")

    if not all_ok:
        print()
        error("Prerequisites not met. Run with --diagnose for details.")
        return 1

    print()

    # Load existing config if present
    existing_env = load_existing_env()
    if existing_env:
        info("Existing .env found — will update (secrets preserved unless you change them)")
        # Mask secrets in display
        display_env = dict(existing_env)
        for secret_key in ["SNITCH_WORKER_TOKEN", "TRANSCRIPTION_API_KEY", "HF_TOKEN", "SNITCH_WORKER_CA_BUNDLE"]:
            if secret_key in display_env:
                display_env[secret_key] = "***" if display_env[secret_key] else ""
        print(f"  Current config: {json.dumps(display_env, indent=2)}")

    # Load defaults from .env.example
    config = load_env_example()

    # Merge existing config
    if existing_env:
        config.update(existing_env)

    # Prompt for configuration
    config = prompt_for_config(config, existing_env)

    # Write .env
    write_env_file(config)

    # Set up models directory
    models_dir = config.get("SNITCH_MODELS_DIR", str(MODELS_DIR_DEFAULT))
    setup_models_dir(models_dir, platform_key, details)

    # Download models if opted in
    if config.get("SNITCH_VOICE_DOWNLOAD_ON_START") == "1":
        hf_token = config.get("HF_TOKEN", "")
        confirm_terms = config.get("SNITCH_VOICE_CONFIRM_TERMS") == "1"
        if hf_token and confirm_terms:
            # Download now (before starting worker) to avoid bootstrap delay
            if not download_models(models_dir, hf_token, confirm_terms):
                warn("Model download failed — worker will start but voice features will be unavailable")
        else:
            warn("Skipping model download (missing HF_TOKEN or terms not confirmed)")

    # Pull latest image
    if not args.no_pull:
        if not docker_compose_pull():
            return 1

    # Start worker
    if not args.no_start:
        if not docker_compose_up(detach=True):
            return 1

    print()
    success("Installation complete!")
    print()
    print("Next steps:")
    print(f"  1. View logs:    {Colors.CYAN}docker compose logs -f worker{Colors.RESET} (in {WORKER_DIR})")
    print(f"  2. Check status: {Colors.CYAN}python {__file__} --diagnose{Colors.RESET}")
    print(f"  3. In Snitch Admin: verify worker appears with capabilities")
    print()
    print("To update later: run this script again with --update")
    return 0

def cmd_update(args: argparse.Namespace) -> int:
    """Update an existing installation."""
    print(bold("=== Snitch Worker Update ==="))
    print()

    if not ENV_FILE.exists():
        error("No existing installation found (.env missing). Run without --update for fresh install.")
        return 1

    existing_env = load_existing_env()
    if not existing_env:
        error("Could not read existing .env")
        return 1

    # Detect platform
    platform_key, details = detect_platform()

    # Check prerequisites
    all_ok, messages = check_prerequisites(platform_key, details)
    for status, msg in messages:
        if status == "ok":
            print(f"  {Colors.GREEN}✓{Colors.RESET} {msg}")
        elif status == "warn":
            print(f"  {Colors.YELLOW}⚠{Colors.RESET} {msg}")
        else:
            print(f"  {Colors.RED}✗{Colors.RESET} {msg}")

    if not all_ok:
        print()
        error("Prerequisites not met.")
        return 1

    # Load defaults and merge with existing (existing wins for secrets)
    config = load_env_example()
    config.update(existing_env)

    # Ask if user wants to change any values
    print()
    print(bold("Current configuration (secrets masked):"))
    display_env = dict(config)
    for secret_key in ["SNITCH_WORKER_TOKEN", "TRANSCRIPTION_API_KEY", "HF_TOKEN", "SNITCH_WORKER_CA_BUNDLE"]:
        if secret_key in display_env:
            display_env[secret_key] = "***" if display_env[secret_key] else ""
    print(json.dumps(display_env, indent=2))

    change = input("\nModify any values? [y/N]: ").strip().lower()
    if change in ("y", "yes", "1", "true"):
        config = prompt_for_config(config, existing_env)
        write_env_file(config)
    else:
        info("Keeping existing configuration")

    # Set up models directory (in case path changed)
    models_dir = config.get("SNITCH_MODELS_DIR", str(MODELS_DIR_DEFAULT))
    setup_models_dir(models_dir, platform_key, details)

    # Pull latest image
    if not args.no_pull:
        if not docker_compose_pull():
            return 1

    # Restart worker
    if not args.no_start:
        info("Restarting worker...")
        run_command(["docker", "compose", "down"], capture=True, cwd=WORKER_DIR)
        if not docker_compose_up(detach=True):
            return 1

    success("Update complete!")
    return 0

def main() -> int:
    parser = argparse.ArgumentParser(
        description="Snitch Worker — Single-Command Installer",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog=textwrap.dedent("""
        Examples:
          python install.py                    # Interactive install
          python install.py --diagnose         # Check prerequisites and config
          python install.py --update           # Update existing installation
          curl -fsSL https://raw.githubusercontent.com/thomas-bsn/snitch/main/deploy/worker/install.py | python3 -
        """),
    )
    parser.add_argument("--diagnose", action="store_true", help="Run diagnostic checks only")
    parser.add_argument("--update", action="store_true", help="Update existing installation (preserves config)")
    parser.add_argument("--no-pull", action="store_true", help="Skip pulling the latest image")
    parser.add_argument("--no-start", action="store_true", help="Configure but don't start the worker")

    args = parser.parse_args()

    if args.diagnose:
        platform_key, details = detect_platform()
        run_diagnostics(platform_key, details)
        return 0

    if args.update:
        return cmd_update(args)

    return cmd_install(args)

if __name__ == "__main__":
    sys.exit(main())