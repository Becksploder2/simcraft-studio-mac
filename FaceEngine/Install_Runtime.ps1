$ErrorActionPreference='Stop'
$root=Split-Path -Parent $PSScriptRoot
$engine=$PSScriptRoot
$runtime=Join-Path $engine 'runtime'
$log=Join-Path $engine 'runtime_install.log'
Start-Transcript -Path $log -Force | Out-Null
try {
  [Net.ServicePointManager]::SecurityProtocol=[Net.SecurityProtocolType]::Tls12
  New-Item -ItemType Directory -Force -Path $runtime | Out-Null
  $zip=Join-Path $env:TEMP 'simmorph_python_embed_3119.zip'
  Invoke-WebRequest 'https://www.python.org/ftp/python/3.11.9/python-3.11.9-embed-amd64.zip' -OutFile $zip -UseBasicParsing
  Expand-Archive -Path $zip -DestinationPath $runtime -Force
  $pth=Join-Path $runtime 'python311._pth'
  @('python311.zip','.','Lib\site-packages','import site') | Set-Content $pth -Encoding ascii
  New-Item -ItemType Directory -Force -Path (Join-Path $runtime 'Lib\site-packages') | Out-Null
  $getpip=Join-Path $runtime 'get-pip.py'
  Invoke-WebRequest 'https://bootstrap.pypa.io/get-pip.py' -OutFile $getpip -UseBasicParsing
  & (Join-Path $runtime 'python.exe') $getpip --no-warn-script-location
  if($LASTEXITCODE -ne 0){throw 'pip bootstrap failed'}
  & (Join-Path $runtime 'python.exe') -m pip install --no-warn-script-location -r (Join-Path $engine 'requirements.txt')
  if($LASTEXITCODE -ne 0){throw 'MediaPipe installation failed'}
  & (Join-Path $runtime 'python.exe') -c 'import mediapipe, PIL; print("MediaPipe runtime ready:", mediapipe.__version__)'
  if($LASTEXITCODE -ne 0){throw 'MediaPipe import verification failed'}
  Write-Host 'SUCCESS: Bundled runtime installed. Place face_landmarker.task in Models and restart SimMorph Studio.'
} catch {
  Write-Host ('INSTALL FAILED: '+$_.Exception.Message)
  Write-Host ('See log: '+$log)
} finally { Stop-Transcript | Out-Null; Read-Host 'Press Enter to close' }
