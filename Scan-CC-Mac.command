#!/bin/bash
cd "$(dirname "$0")"
DEFAULT="$HOME/Documents/Electronic Arts/The Sims 4/Mods"
read -r -p "Mods folder [$DEFAULT]: " MODS
python3 CCScanner.py "${MODS:-$DEFAULT}" --hash
read -r -p "Press Enter to close..." _
