param([Parameter(Mandatory=$true)][string]$OutputPath)
$ErrorActionPreference='Stop'
Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing
Add-Type @'
using System;
using System.Runtime.InteropServices;
using System.Text;
public class CASWindow {
 [DllImport("user32.dll")] public static extern IntPtr GetForegroundWindow();
 [DllImport("user32.dll", CharSet=CharSet.Unicode)] public static extern int GetWindowText(IntPtr h, StringBuilder s, int n);
 [DllImport("user32.dll")] public static extern bool GetWindowRect(IntPtr h, out RECT r);
 [StructLayout(LayoutKind.Sequential)] public struct RECT { public int Left,Top,Right,Bottom; }
}
'@
$hwnd=[CASWindow]::GetForegroundWindow()
$sb=New-Object System.Text.StringBuilder 512
[void][CASWindow]::GetWindowText($hwnd,$sb,$sb.Capacity)
if ($sb.ToString() -notmatch '(?i)sims\s*4|the sims') { throw "The Sims 4 must be the active window at capture time. Foreground: $($sb.ToString())" }
$rect=New-Object CASWindow+RECT
if (-not [CASWindow]::GetWindowRect($hwnd,[ref]$rect)) { throw 'Unable to read CAS window bounds' }
$w=$rect.Right-$rect.Left; $h=$rect.Bottom-$rect.Top
if($w -lt 100 -or $h -lt 100){throw 'CAS window dimensions invalid'}
$bmp=New-Object System.Drawing.Bitmap($w,$h)
try {
 $g=[System.Drawing.Graphics]::FromImage($bmp)
 try {$g.CopyFromScreen($rect.Left,$rect.Top,0,0,(New-Object System.Drawing.Size($w,$h)))} finally {$g.Dispose()}
 $bmp.Save($OutputPath,[System.Drawing.Imaging.ImageFormat]::Png)
} finally {$bmp.Dispose()}
