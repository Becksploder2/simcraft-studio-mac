@echo off
set /p PHOTO=Enter full path to reference JPG/PNG: 
python "%~dp0analyze_faces.py" "%PHOTO%" --cas "%~dp0..\Captures\current_cas.png" --output "%~dp0..\Captures\face_analysis.json"
pause
