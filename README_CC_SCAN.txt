SIMCRAFT STUDIO 5.1 — CC INVENTORY (WINDOWS / MAC)

Requires Python 3.9+ installed on your computer.
Windows: extract ZIP, double-click Scan-CC-Windows.bat.
Mac: extract ZIP, open Terminal, run: bash Scan-CC-Mac.command
Or use: python CCScanner.py "path/to/The Sims 4/Mods" --hash

The scanner reads package headers, inventories all .package files recursively,
flags invalid DBPF signatures, and optionally detects exact duplicate files.
It saves Simcraft-CC-Catalog.json in the working directory. Open the app's
CC Library tab and import that JSON file. Categories are GUESSES based only
on filenames. Unknown items stay unknown. It does not yet inspect CAS resources,
identify swatches/meshes, or extract thumbnails.

The scanner does NOT change, delete, move or write to any package or Mods folder.
The inventory contains local relative paths; don't share it publicly if you
want to keep your CC collection private.

The Windows application is the existing v5.0 executable paired with an updated
v5.1 Studio.html workspace. The scanner runs separately, not inside the EXE.
Mac remains a source/build kit and must be compiled on macOS.
