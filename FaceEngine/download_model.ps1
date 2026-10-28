$ErrorActionPreference='Stop'
$target=Join-Path $PSScriptRoot '..\Models\face_landmarker.task'
$url='https://storage.googleapis.com/mediapipe-models/face_landmarker/face_landmarker/float16/latest/face_landmarker.task'
Invoke-WebRequest -Uri $url -OutFile $target
Write-Host "Saved $target"
