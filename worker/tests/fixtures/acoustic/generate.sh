#!/usr/bin/env bash
# Dedicated synthetic French voice, no recording or library content.
# espeak-ng 1.52, GPL-3 software; generated speech data is unrestricted.
# Run explicitly to regenerate; normal tests never invoke TTS.
set -euo pipefail
cd "$(dirname "$0")"
espeak-ng -v fr-fr -s 145 -w voice-source.wav 'Bonjour, ceci est une voix artificielle pour vérifier la présence de parole. Le personnage du jeu parle aussi dans cette scène.'
ffmpeg -v error -y -i voice-source.wav -ar 16000 -c:a pcm_s16le voice.wav
rm voice-source.wav
