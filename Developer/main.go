//go:build windows
package main

import (
 "embed"
 "io"
 "context"
 "encoding/json"
 "fmt"
 "image"
 "image/color"
 "image/png"
 "encoding/binary"

 _ "image/jpeg"
 _ "image/png"
 "math"
 "strconv"
 "os/exec"
 
 "runtime"
 "time"
  "strings"
 "sync/atomic"
 "sync"
 "os"
 "path/filepath"
 "syscall"
 "unsafe"
)

//go:embed assets/cas_room.bmp
var assets embed.FS
var (user32=syscall.NewLazyDLL("user32.dll");gdi32=syscall.NewLazyDLL("gdi32.dll");comdlg=syscall.NewLazyDLL("comdlg32.dll");kernel=syscall.NewLazyDLL("kernel32.dll")
 regClass=user32.NewProc("RegisterClassExW");createWin=user32.NewProc("CreateWindowExW");defProc=user32.NewProc("DefWindowProcW");showWin=user32.NewProc("ShowWindow");updateWin=user32.NewProc("UpdateWindow");getMsg=user32.NewProc("GetMessageW");transMsg=user32.NewProc("TranslateMessage");dispatch=user32.NewProc("DispatchMessageW");sendMsg=user32.NewProc("SendMessageW");setText=user32.NewProc("SetWindowTextW");getText=user32.NewProc("GetWindowTextW");getDC=user32.NewProc("GetDC");beginPaint=user32.NewProc("BeginPaint");endPaint=user32.NewProc("EndPaint");releaseDC=user32.NewProc("ReleaseDC");invalidate=user32.NewProc("InvalidateRect");loadImage=user32.NewProc("LoadImageW");getObj=gdi32.NewProc("GetObjectW");createDC=gdi32.NewProc("CreateCompatibleDC");selectObj=gdi32.NewProc("SelectObject");stretch=gdi32.NewProc("StretchBlt");deleteDC=gdi32.NewProc("DeleteDC");deleteObj=gdi32.NewProc("DeleteObject");setBkMode=gdi32.NewProc("SetBkMode");setColor=gdi32.NewProc("SetTextColor");textOut=gdi32.NewProc("TextOutW");getOpen=comdlg.NewProc("GetOpenFileNameW");getSave=comdlg.NewProc("GetSaveFileNameW");getModule=kernel.NewProc("GetModuleHandleW");shell32=syscall.NewLazyDLL("shell32.dll");dragAccept=shell32.NewProc("DragAcceptFiles");dragQuery=shell32.NewProc("DragQueryFileW");dragFinish=shell32.NewProc("DragFinish");postMessage=user32.NewProc("PostMessageW")
 mainHWND uintptr;bgBmp uintptr;previewBmp uintptr;referenceBmp uintptr;previewMu sync.Mutex;previewPathMu sync.Mutex;pendingPreviewPath,pendingReferencePath string;captureBusy atomic.Bool; photo string;gender="F";selected=1; labels=[]string{"Face width","Face height","Eye spacing","Eye size","Nose width","Nose length","Jaw width","Chin length","Mouth width","Lip fullness"}; values=[10]int{50,50,50,50,50,50,50,50,50,50};controls [10]uintptr;templateLabel,statusLabel,photoLabel,folderLabel,autoTemplateLabel uintptr; modeLabel,libraryLabel,pathField,modelLabel,gameLabel uintptr; importBusy atomic.Bool; importResults=make(chan photoResult,2);dialogResults=make(chan string,2);dialogBusy atomic.Bool; libraryMu sync.RWMutex; libraryCount int; libraryReady bool; casX,casY int; casCalibrated bool; statusMu sync.Mutex; pendingStatus string
)
const (WS_OVERLAPPEDWINDOW=0x00CF0000;WS_VISIBLE=0x10000000;WS_CHILD=0x40000000;WS_BORDER=0x00800000;ES_READONLY=0x800;CBS_DROPDOWNLIST=3;WM_DESTROY=2;WM_COMMAND=0x111;WM_PAINT=15;WM_ERASEBKGND=20;WM_CTLCOLORSTATIC=0x138;WM_HSCROLL=0x114;WM_DROPFILES=0x233;WM_PHOTO_READY=0x8001;WM_DIALOG_READY=0x8002;WM_LIBRARY_READY=0x8003;BM_GETCHECK=0xF0;SRCCOPY=0xCC0020)
type wc struct {Size uint32;Style uint32;WndProc uintptr;ClsExtra int32;WndExtra int32;Instance uintptr;Icon uintptr;Cursor uintptr;Background uintptr;MenuName *uint16;ClassName *uint16;IconSm uintptr}
type msg struct {Hwnd uintptr;Message uint32;Wparam uintptr;Lparam uintptr;Time uint32;PtX int32;PtY int32;Private uint32}
type paintstruct struct {Hdc uintptr;Erase int32;Rc [4]int32;Restore int32;IncUpdate int32;Reserved [32]byte}
type bitmap struct {Type int32;Width int32;Height int32;WidthBytes int32;Planes uint16;BitsPixel uint16;Bits uintptr}
type ofn struct {Size uint32;Owner uintptr;Instance uintptr;Filter *uint16;CustomFilter uintptr;MaxCustFilter uint32;FilterIndex uint32;File *uint16;MaxFile uint32;FileTitle uintptr;MaxFileTitle uint32;InitialDir uintptr;Title *uint16;Flags uint32;FileOffset uint16;FileExtension uint16;DefExt uintptr;CustData uintptr;Hook uintptr;TemplateName uintptr;ReservedPtr uintptr;Reserved uint32;FlagsEx uint32}
func ptr(s string)*uint16{return syscall.StringToUTF16Ptr(s)}
func win(class,name string,style uint32,x,y,w,h int,parent uintptr,id int)uintptr{v,_,_:=createWin.Call(0,uintptr(unsafe.Pointer(ptr(class))),uintptr(unsafe.Pointer(ptr(name))),uintptr(style),uintptr(x),uintptr(y),uintptr(w),uintptr(h),parent,uintptr(id),0,0);return v}
func label(text string,x,y,w,h int)uintptr{return win("STATIC",text,WS_VISIBLE|WS_CHILD,x,y,w,h,mainHWND,0)}
func button(text string,x,y,w,h,id int)uintptr{return win("BUTTON",text,WS_VISIBLE|WS_CHILD,x,y,w,h,mainHWND,id)}
func change(h uintptr,s string){setText.Call(h,uintptr(unsafe.Pointer(ptr(s))))}
func fileDialog(save bool)string{b:=make([]uint16,1024);filter:=syscall.StringToUTF16("Image files (*.png;*.jpg;*.jpeg)\x00*.png;*.jpg;*.jpeg\x00All files\x00*.*\x00\x00");if save{filter=syscall.StringToUTF16("JSON files (*.json)\x00*.json\x00\x00")};d:=ofn{Size:uint32(unsafe.Sizeof(ofn{})),Owner:mainHWND,Filter:&filter[0],File:&b[0],MaxFile:1024,Flags:0x80000|0x1000};if save{d.Title=ptr("Export CAS guide");d.Flags=0x80000|0x2;d.DefExt=uintptr(unsafe.Pointer(ptr("json")));d.File= &b[0];copy(b,syscall.StringToUTF16("SimMorph_CAS_Guide.json"))};var r uintptr;if save{r,_,_=getSave.Call(uintptr(unsafe.Pointer(&d)))}else{r,_,_=getOpen.Call(uintptr(unsafe.Pointer(&d)))};if r==0{return ""};return syscall.UTF16ToString(b)}
type photoResult struct {path string; width,height int; err string; suggestion string; matchScore float64; matchNote string}

// estimateTemplate is a deliberately lightweight, offline visual ranking.
// It does NOT detect face landmarks or infer a person's gender. It compares
// normalized grayscale structure and gradient magnitude in a central portrait
// crop. Scores are similarity heuristics, not probabilities of correctness.
func templateFeatures(img image.Image) []float64 {
 b:=img.Bounds(); w,h:=b.Dx(),b.Dy()
 const n=18
 grid:=make([]float64,n*n)
 for y:=0;y<n;y++ {for x:=0;x<n;x++ {
   // Center crop: avoid outermost edges, which often contain backgrounds.
   xx:=b.Min.X+int((0.16+0.68*(float64(x)+0.5)/n)*float64(w-1))
   yy:=b.Min.Y+int((0.08+0.78*(float64(y)+0.5)/n)*float64(h-1))
   r,g,bl,_:=img.At(xx,yy).RGBA()
   grid[y*n+x]=(0.2126*float64(r)+0.7152*float64(g)+0.0722*float64(bl))/65535
 }}
 mean:=0.0;for _,v:=range grid{mean+=v};mean/=float64(len(grid))
 variance:=0.0;for _,v:=range grid{variance+=(v-mean)*(v-mean)}
 std:=math.Sqrt(variance/float64(len(grid)));if std<0.05{std=0.05}
 for i,v:=range grid{grid[i]=(v-mean)/std}
 feat:=make([]float64,0,2*(n-2)*(n-2))
 for y:=1;y<n-1;y++{for x:=1;x<n-1;x++{
   dx:=grid[y*n+x+1]-grid[y*n+x-1]
   dy:=grid[(y+1)*n+x]-grid[(y-1)*n+x]
   feat=append(feat,math.Abs(dx),math.Abs(dy))
 }}
 return feat
}
func estimateTemplate(path string)(string,float64,string){
 f,err:=os.Open(path);if err!=nil{return "",0,err.Error()}
 defer f.Close()
 im,_,err:=image.Decode(f);if err!=nil{return "",0,"image decode failed"}
 ref:=templateFeatures(im)
 best:="";bestDist:=math.MaxFloat64
 for _,g:=range []string{"F","M"} {
   folder:="Female";if g=="M"{folder="Male"}
   for i:=1;i<=19;i++ {
     name:=fmt.Sprintf("%s%d",g,i)
     file,err:=os.Open(filepath.Join(appRoot(),"Assets","FaceTemplates",folder,name+".png"))
     if err!=nil{continue}
     tmpl,_,err:=image.Decode(file);file.Close();if err!=nil{continue}
     feat:=templateFeatures(tmpl);sum:=0.0
     for j,v:=range feat{d:=v-ref[j];sum+=d*d}
     distance:=math.Sqrt(sum/float64(len(feat)))
     if distance<bestDist{bestDist=distance;best=name}
   }
 }
 if best==""{return "",0,"template images not found in Assets/FaceTemplates"}
 // Normalized visual score, NOT confidence. Avoid implying accuracy.
 score:=math.Max(0,math.Min(1,1-bestDist/3))
 return best,score,""
}
func loadPhoto(path string){
 if !importBusy.CompareAndSwap(false,true){change(statusLabel,"An image is already loading. Please wait.");return}
 if ext:=strings.ToLower(filepath.Ext(path));ext!=".png"&&ext!=".jpg"&&ext!=".jpeg" {importBusy.Store(false);change(statusLabel,"Please use a JPG or PNG image.");return}
 change(statusLabel,"Reading image information...")
 go func(){
  result:=photoResult{path:path}
  defer func(){if recover()!=nil{result.err="Unexpected error reading image"};importResults<-result;postMessage.Call(mainHWND,WM_PHOTO_READY,0,0)}()
  f,e:=os.Open(path);if e!=nil{result.err=e.Error();return};defer f.Close()
  info,e:=f.Stat();if e!=nil{result.err=e.Error();return};if info.Size()>35*1024*1024{result.err="Image exceeds 35 MB limit";return}
  config,_,e:=image.DecodeConfig(f);if e!=nil{result.err="Unsupported or damaged image";return}
  if config.Width<50||config.Height<50||int64(config.Width)*int64(config.Height)>40000000{result.err="Image dimensions outside supported range";return}
  result.width,result.height=config.Width,config.Height
  result.suggestion,result.matchScore,result.matchNote=estimateTemplate(path)
 }()
}
func finishPhoto(){result:=<-importResults;importBusy.Store(false);if result.err!=""{change(statusLabel,result.err);return};photo=result.path;change(photoLabel,filepath.Base(photo));updateGuide(result.width,result.height);refreshReference();if result.suggestion!=""{gender=result.suggestion[:1];selected,_=strconv.Atoi(result.suggestion[1:]);change(templateLabel,"Selected: "+result.suggestion);change(autoTemplateLabel,fmt.Sprintf("Experimental suggestion: %s (visual score %.0f%%) — verify manually",result.suggestion,result.matchScore*100));change(statusLabel,"Auto-selected "+result.suggestion+". Visual similarity only; not validated face landmarks. Click Female/Male Templates to override.")}else{change(autoTemplateLabel,"Auto selection unavailable: "+result.matchNote)}}

func browsePhoto(){
 if !dialogBusy.CompareAndSwap(false,true){return}
 change(statusLabel,"Opening Windows photo browser...")
 go func(){
  runtime.LockOSThread()
  defer runtime.UnlockOSThread()
  ole:=syscall.NewLazyDLL("ole32.dll")
  coInit:=ole.NewProc("CoInitializeEx")
  coUninit:=ole.NewProc("CoUninitialize")
  hr,_,_:=coInit.Call(0,2) // COINIT_APARTMENTTHREADED
  if hr==0 || hr==1 {defer coUninit.Call()}
  result:=""
  defer func(){
   if r:=recover();r!=nil {logError(fmt.Sprintf("Picker panic: %v",r))}
   dialogResults<-result
   postMessage.Call(mainHWND,WM_DIALOG_READY,0,0)
  }()
  b:=make([]uint16,32768)
  filter:=syscall.StringToUTF16("Photos (*.jpg;*.jpeg;*.png)\\x00*.jpg;*.jpeg;*.png\\x00All files (*.*)\\x00*.*\\x00\\x00")
  title:=ptr("Choose a photo - Simscraft Studio")
  // Do not use the Go window's HWND as owner from a different OS thread.
  d:=ofn{Size:uint32(unsafe.Sizeof(ofn{})),Owner:0,Filter:&filter[0],File:&b[0],MaxFile:uint32(len(b)),Title:title,Flags:0x00080000|0x00001000|0x00000008}
  r,_,err:=getOpen.Call(uintptr(unsafe.Pointer(&d)))
  runtime.KeepAlive(filter);runtime.KeepAlive(title);runtime.KeepAlive(b)
  if r==0 {logError(fmt.Sprintf("GetOpenFileNameW failed or cancelled: %v",err));return}
  result=syscall.UTF16ToString(b)
 }()
}
func finishBrowse(){
 dialogBusy.Store(false)
 path:=<-dialogResults
 if path=="" {change(statusLabel,"No photo selected. If the dialog failed, see Logs/app.log.");return}
 loadPhoto(path)
}
func logError(s string){
 root:=filepath.Join(os.Getenv("LOCALAPPDATA"),"SimMorphStudio","Logs")
 if os.Getenv("LOCALAPPDATA")=="" {root=filepath.Join(os.TempDir(),"SimMorphStudioLogs")}
 _=os.MkdirAll(root,0755)
 f,e:=os.OpenFile(filepath.Join(root,"app.log"),os.O_CREATE|os.O_APPEND|os.O_WRONLY,0644)
 if e==nil {defer f.Close();fmt.Fprintf(f,"%s %s\\n",time.Now().Format(time.RFC3339),s)}
}
// CAS bridge v1: capture an on-screen preset tile and activate only when Sims 4 is foreground.
var registerHotKey=user32.NewProc("RegisterHotKey")
var unregisterHotKey=user32.NewProc("UnregisterHotKey")
var getCursorPos=user32.NewProc("GetCursorPos")
var setCursorPos=user32.NewProc("SetCursorPos")
var mouseEvent=user32.NewProc("mouse_event")
var foreground=user32.NewProc("GetForegroundWindow")
var windowText=user32.NewProc("GetWindowTextW")
type point struct{X,Y int32}
func calibrateCAS(){
 change(statusLabel,"Move cursor over desired CAS preset tile within 4 seconds...")
 go func(){
  time.Sleep(4*time.Second)
  var pt point
  r,_,_:=getCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
  if r==0 {logError("Unable to capture CAS cursor position");return}
  casX,casY=int(pt.X),int(pt.Y);casCalibrated=true
  saveCASCalibration()
  postMessage.Call(mainHWND,0x8004,0,0)
 }()
}
var applyBusy atomic.Bool
func applyCAS(){
 if !casCalibrated {change(statusLabel,"Calibrate first: click 'Set CAS Click Point', then hover over the desired in-game control.");return}
 if !applyBusy.CompareAndSwap(false,true){change(statusLabel,"An in-game click is already pending.");return}
 change(statusLabel,"In 5 seconds: one click at saved CAS point. Switch to Sims 4 and keep CAS open. ESC cancels by switching windows.")
 go func(){
  defer applyBusy.Store(false)
  time.Sleep(5*time.Second)
  if !isSimsForeground(){postStatus("CANCELLED: Sims 4 was not the foreground window. No click sent.");return}
  setCursorPos.Call(uintptr(casX),uintptr(casY))
  mouseEvent.Call(0x0002,0,0,0,0)
  mouseEvent.Call(0x0004,0,0,0,0)
  postStatus("CAS click sent. Capturing the screen in 2 seconds; this does not prove a face morph occurred.")
  time.Sleep(2*time.Second)
  if !isSimsForeground(){postStatus("CAS click sent. Return to Sims 4 to inspect the result; no screenshot taken.");return}
  out:=filepath.Join(appRoot(),"Captures","current_cas.png")
  _=os.MkdirAll(filepath.Dir(out),0755)
  if err:=nativeScreenCapture(out);err!=nil {postStatus("CAS click sent, but verification screenshot failed: "+err.Error());return}
  postStatus("CAS click sent; fresh screenshot saved. Inspect the in-game Sim to confirm the change.")
  refreshPreview()
 }()
}
func saveCASCalibration(){
 cfg:=filepath.Join(appRoot(),"Config","cas_calibration.json")
 _=os.MkdirAll(filepath.Dir(cfg),0755)
 b,_:=json.MarshalIndent(map[string]interface{}{"x":casX,"y":casY,"calibrated":casCalibrated,"selected_template":gender+strconv.Itoa(selected)},"","  ")
 if e:=os.WriteFile(cfg,b,0644);e!=nil{logError("CAS calibration save: "+e.Error())}
}
// Model manager intentionally does not modify the proven v2.0 photo importer.
func modelDir()string{return filepath.Join(appRoot(),"Models")}
func modelStatus()string{
 entries,err:=os.ReadDir(modelDir());if err!=nil{return "Face model: not installed (Models folder missing)"}
 for _,e:=range entries{if e.IsDir(){continue};name:=strings.ToLower(e.Name());if name=="face_landmarker.task"{info,err:=e.Info();if err==nil&&info.Size()>1024{return fmt.Sprintf("Face landmark model: file detected (%d KB); runtime not integrated",info.Size()/1024)}}}
 return "Face landmark model: not installed (face_landmarker.task needed)"
}
func refreshModel(){change(modelLabel,modelStatus());change(statusLabel,"Model scan complete. Detection requires a compatible inference runtime.")}
func openModels(){_ = os.MkdirAll(modelDir(),0755);cmd:=exec.Command("explorer.exe",modelDir());cmd.SysProcAttr=&syscall.SysProcAttr{HideWindow:true,CreationFlags:0x08000000};if err:=cmd.Start();err!=nil{change(statusLabel,"Could not open Models folder: "+err.Error())}}
var bitBlt=gdi32.NewProc("BitBlt")
var getDIBits=gdi32.NewProc("GetDIBits")
var createBitmap=gdi32.NewProc("CreateCompatibleBitmap")
var getSystemMetrics=user32.NewProc("GetSystemMetrics")
type bmpInfoHeader struct{Size uint32;Width,Height int32;Planes,BitCount uint16;Compression,SizeImage uint32;XPelsPerMeter,YPelsPerMeter int32;ClrUsed,ClrImportant uint32}
type bmpInfo struct{Header bmpInfoHeader;Colors [4]byte}
func nativeScreenCapture(path string)error{
 wv,_,_:=getSystemMetrics.Call(0);hv,_,_:=getSystemMetrics.Call(1)
 w,h:=int(wv),int(hv);if w<=0||h<=0||w>16000||h>16000{return fmt.Errorf("invalid display dimensions")}
 dc,_,_:=getDC.Call(0);if dc==0{return fmt.Errorf("GetDC failed")};defer releaseDC.Call(0,dc)
 mem,_,_:=createDC.Call(dc);if mem==0{return fmt.Errorf("CreateCompatibleDC failed")};defer deleteDC.Call(mem)
 bmp,_,_:=createBitmap.Call(dc,uintptr(w),uintptr(h));if bmp==0{return fmt.Errorf("CreateCompatibleBitmap failed")};defer deleteObj.Call(bmp)
 old,_,_:=selectObj.Call(mem,bmp);defer selectObj.Call(mem,old)
 ok,_,_:=bitBlt.Call(mem,0,0,uintptr(w),uintptr(h),dc,0,0,SRCCOPY);if ok==0{return fmt.Errorf("BitBlt failed")}
 bi:=bmpInfo{Header:bmpInfoHeader{Size:40,Width:int32(w),Height:-int32(h),Planes:1,BitCount:32}}
 pix:=make([]byte,w*h*4)
 got,_,_:=getDIBits.Call(mem,bmp,0,uintptr(h),uintptr(unsafe.Pointer(&pix[0])),uintptr(unsafe.Pointer(&bi)),0)
 if got==0{return fmt.Errorf("GetDIBits failed")}
 img:=image.NewRGBA(image.Rect(0,0,w,h))
 for i:=0;i<len(pix);i+=4{img.Pix[i],img.Pix[i+1],img.Pix[i+2],img.Pix[i+3]=pix[i+2],pix[i+1],pix[i],255}
 f,err:=os.Create(path);if err!=nil{return err};defer f.Close();return png.Encode(f,img)
}
func saveBMPPreview(src string) (string,error){
 f,e:=os.Open(src);if e!=nil{return "",e};defer f.Close()
 im,_,e:=image.Decode(f);if e!=nil{return "",e}
 // Render into a fixed-size bitmap for the Win32 preview panel.
 const w,h=900,650
 canvas:=image.NewRGBA(image.Rect(0,0,w,h));for y:=0;y<h;y++{for x:=0;x<w;x++{canvas.Set(x,y,color.RGBA{30,32,38,255})}}
 bounds:=im.Bounds();sw,sh:=bounds.Dx(),bounds.Dy();if sw<1||sh<1{return "",fmt.Errorf("empty image")}
 scale:=math.Min(float64(w)/float64(sw),float64(h)/float64(sh));dw,dh:=int(float64(sw)*scale),int(float64(sh)*scale)
 ox,oy:=(w-dw)/2,(h-dh)/2
 for y:=0;y<dh;y++{for x:=0;x<dw;x++{sx:=bounds.Min.X+int(float64(x)/scale);sy:=bounds.Min.Y+int(float64(y)/scale);canvas.Set(ox+x,oy+y,im.At(sx,sy))}}
 out:=filepath.Join(os.TempDir(),"simmorph_cas_preview.bmp");if src==photo {out=filepath.Join(os.TempDir(),"simmorph_reference_preview.bmp")}
 bf,e:=os.Create(out);if e!=nil{return "",e};defer bf.Close()
 rowSize:=(w*3+3)&^3;size:=rowSize*h
 header:=make([]byte,54);copy(header[:2],[]byte("BM"));binary.LittleEndian.PutUint32(header[2:],uint32(54+size));binary.LittleEndian.PutUint32(header[10:],54);binary.LittleEndian.PutUint32(header[14:],40);binary.LittleEndian.PutUint32(header[18:],w);binary.LittleEndian.PutUint32(header[22:],h);binary.LittleEndian.PutUint16(header[26:],1);binary.LittleEndian.PutUint16(header[28:],24);binary.LittleEndian.PutUint32(header[34:],uint32(size));if _,e=bf.Write(header);e!=nil{return "",e}
 row:=make([]byte,rowSize);for y:=h-1;y>=0;y--{for x:=0;x<w;x++{i:=y*canvas.Stride+x*4;row[x*3],row[x*3+1],row[x*3+2]=canvas.Pix[i+2],canvas.Pix[i+1],canvas.Pix[i]};if _,e=bf.Write(row);e!=nil{return "",e}}
 return out,nil
}
func refreshReference(){
 if photo=="" {return}
 current:=photo
 go func(){
  file,err:=saveBMPPreview(current)
  if err!=nil {logError("Reference preview: "+err.Error());return}
  previewPathMu.Lock();pendingReferencePath=file;previewPathMu.Unlock()
  postMessage.Call(mainHWND,0x8008,0,0)
 }()
}
func refreshPreview(){
 path:=filepath.Join(appRoot(),"Captures","current_cas.png")
 if _,err:=os.Stat(path);err!=nil{postStatus("No CAS screenshot yet. Capture the current Sim first.");return}
 go func(){
  file,err:=saveBMPPreview(path)
  if err!=nil {logError("Preview: "+err.Error());postStatus("Could not render CAS preview.");return}
  previewPathMu.Lock();pendingPreviewPath=file;previewPathMu.Unlock()
  postMessage.Call(mainHWND,0x8009,0,0)
 }()
}
func loadPreviewOnUI(reference bool){
 previewPathMu.Lock()
 path:=""
 if reference {path=pendingReferencePath;pendingReferencePath=""} else {path=pendingPreviewPath;pendingPreviewPath=""}
 previewPathMu.Unlock()
 if path=="" {return}
 u:=ptr(path)
 bmp,_,_:=loadImage.Call(0,uintptr(unsafe.Pointer(u)),0,0,0,0x10)
 runtime.KeepAlive(u)
 if bmp==0 {logError("LoadImageW failed for "+path);postStatus("Preview load failed; see app.log");return}
 previewMu.Lock()
 var old uintptr
 if reference {old=referenceBmp;referenceBmp=bmp} else {old=previewBmp;previewBmp=bmp}
 previewMu.Unlock()
 if old!=0 {deleteObj.Call(old)}
 invalidate.Call(mainHWND,0,0)
 if !reference {postStatus("CAS screenshot loaded into preview at "+time.Now().Format("15:04:05"))}
}
var themeNames []string
var activeTheme="Default CAS Room"
var themeMenu uintptr
const themeStart=3100
const themeImport=3200
const themeRefresh=3201
func themesDir()string{return filepath.Join(appRoot(),"Backgrounds")}
func readThemeSelection()string{b,e:=os.ReadFile(filepath.Join(appRoot(),"Config","selected_background.txt"));if e!=nil{return "Default CAS Room"};return strings.TrimSpace(string(b))}
func themePath(name string)string{
 if name=="Default CAS Room" {return filepath.Join(appRoot(),"assets","cas_room.bmp")}
 return filepath.Join(themesDir(),name)
}
func themeList()[]string{
 names:=[]string{"Default CAS Room"}
 entries,e:=os.ReadDir(themesDir());if e!=nil{return names}
 for _,entry:=range entries{if entry.IsDir(){continue};ext:=strings.ToLower(filepath.Ext(entry.Name()));if ext==".bmp"||ext==".png"||ext==".jpg"||ext==".jpeg"{names=append(names,entry.Name())}}
 return names
}
func loadBackgroundOnUI(){
 path:=themePath(activeTheme)
 u:=ptr(path)
 bmp,_,_:=loadImage.Call(0,uintptr(unsafe.Pointer(u)),0,0,0,0x10)
 runtime.KeepAlive(u)
 if bmp==0 {logError("CAS background could not be loaded: "+path);return}
 old:=bgBmp;bgBmp=bmp;if old!=0{deleteObj.Call(old)}
 invalidate.Call(mainHWND,0,0)
}
func captureCurrentCAS(){
 if !captureBusy.CompareAndSwap(false,true){postStatus("CAS capture already in progress.");return}
 postStatus("CAS capture in 3 seconds. Switch to Sims 4 now...")
 go func(){defer captureBusy.Store(false);time.Sleep(3*time.Second)
  out:=filepath.Join(appRoot(),"Captures","current_cas.png");_ =os.MkdirAll(filepath.Dir(out),0755)
  if err:=nativeScreenCapture(out);err!=nil{logError("Native CAS capture: "+err.Error());postStatus("Capture failed: "+err.Error());return}
  postStatus("CAS screenshot captured. Updating preview...");refreshPreview()
 }()
}
func postStatus(s string){
 // Status updates are delivered to the GUI thread via a custom Windows message.
 statusMu.Lock();pendingStatus=s;statusMu.Unlock()
 postMessage.Call(mainHWND,0x8005,0,0)
}
// Calibrated preset grid: the user selects a template ID; no photo matching is claimed.
var gridFirst,gridLast point
var gridFirstSet,gridLastSet bool
func captureGridPoint(first bool){
 if first {postStatus("Move pointer to CENTER of CAS preset #1 (top-left). Capturing in 4 seconds.")} else {postStatus("Move pointer to CENTER of CAS preset #18 (third row, sixth column). Capturing in 4 seconds.")}
 go func(){
  time.Sleep(4*time.Second)
  if !isSimsForeground(){postStatus("Calibration stopped: The Sims 4 must be active.");return}
  var pt point
  if r,_,_:=getCursorPos.Call(uintptr(unsafe.Pointer(&pt)));r==0{postStatus("Cannot read pointer position.");return}
  if first {gridFirst=pt;gridFirstSet=true;postStatus(fmt.Sprintf("Preset #1 calibrated at %d,%d",pt.X,pt.Y))} else {gridLast=pt;gridLastSet=true;postStatus(fmt.Sprintf("Preset #18 calibrated at %d,%d",pt.X,pt.Y))}
  saveGrid()
 }()
}
func isSimsForeground()bool{
 hwnd,_,_:=foreground.Call(); if hwnd==0{return false}
 buf:=make([]uint16,256);windowText.Call(hwnd,uintptr(unsafe.Pointer(&buf[0])),uintptr(len(buf)))
 title:=strings.ToLower(syscall.UTF16ToString(buf));return strings.Contains(title,"the sims") || strings.Contains(title,"sims 4")
}
func saveGrid(){
 path:=filepath.Join(appRoot(),"Config","preset_grid.json");_ =os.MkdirAll(filepath.Dir(path),0755)
 b,_:=json.MarshalIndent(map[string]interface{}{"first":gridFirst,"last":gridLast,"first_set":gridFirstSet,"last_set":gridLastSet},"","  ");_ =os.WriteFile(path,b,0644)
}
func loadGrid(){
 b,e:=os.ReadFile(filepath.Join(appRoot(),"Config","preset_grid.json"));if e!=nil{return}
 var v struct{First point `json:"first"`;Last point `json:"last"`;FirstSet bool `json:"first_set"`;LastSet bool `json:"last_set"`}
 if json.Unmarshal(b,&v)==nil {gridFirst=v.First;gridLast=v.Last;gridFirstSet=v.FirstSet;gridLastSet=v.LastSet}
}
func applySelectedPreset(){
 if !gridFirstSet || !gridLastSet {change(statusLabel,"Calibrate preset #1 and #18 before applying.");return}
 n:=selected
 if n==19 {change(statusLabel,"Preset #19 is on the fourth row; this grid calibration only covers #1-18. No click sent.");return}
 x:=float64(gridFirst.X)+float64((n-1)%6)*(float64(gridLast.X-gridFirst.X)/5)
 y:=float64(gridFirst.Y)+float64((n-1)/6)*(float64(gridLast.Y-gridFirst.Y)/2)
 change(statusLabel,fmt.Sprintf("Applying %s%d in 4 seconds. Switch to Sims 4 CAS and keep its preset grid open.",gender,n))
 go func(){
  time.Sleep(4*time.Second)
  if !isSimsForeground(){postStatus("No click sent: Sims 4 is not active.");return}
  setCursorPos.Call(uintptr(int32(math.Round(x))),uintptr(int32(math.Round(y))))
  mouseEvent.Call(0x0002,0,0,0,0);mouseEvent.Call(0x0004,0,0,0,0)
  postStatus(fmt.Sprintf("Clicked calibrated preset %s%d. Verify it in CAS; photo matching is not implemented.",gender,n))
 }()
}
// Native Toolhelp process enumeration: no tasklist.exe or CMD window.
type processEntry struct {
 Size uint32
 Usage uint32
 ProcessID uint32
 DefaultHeapID uintptr
 ModuleID uint32
 Threads uint32
 ParentProcessID uint32
 PriorityClassBase int32
 Flags uint32
 ExeFile [260]uint16
}
var snapshotProc=kernel.NewProc("CreateToolhelp32Snapshot")
var processFirst=kernel.NewProc("Process32FirstW")
var processNext=kernel.NewProc("Process32NextW")
var closeHandle=kernel.NewProc("CloseHandle")
func simsRunning()bool{
 h,_,_:=snapshotProc.Call(0x2,0)
 if h==0 || h==^uintptr(0){return false}
 defer closeHandle.Call(h)
 var entry processEntry
 entry.Size=uint32(unsafe.Sizeof(entry))
 ok,_,_:=processFirst.Call(h,uintptr(unsafe.Pointer(&entry)))
 for ok!=0 {
  name:=strings.ToLower(syscall.UTF16ToString(entry.ExeFile[:]))
  if name=="ts4_x64.exe" || name=="ts4_dx9_x64.exe" {return true}
  ok,_,_=processNext.Call(h,uintptr(unsafe.Pointer(&entry)))
 }
 return false
}
func startGameMonitor(){
 go func(){
  time.Sleep(1500*time.Millisecond)
  previous:=""
  for {
   state:="Sims 4: not running"
   if simsRunning(){state="Sims 4: running (CAS not verified)"}
   if state!=previous {
    gameMu.Lock();gameText=state;gameMu.Unlock()
    if mainHWND!=0 {postMessage.Call(mainHWND,0x8006,0,0)}
    previous=state
   }
   time.Sleep(4*time.Second)
  }
 }()
}
var gameMu sync.Mutex
var gameText string
func gameStatus()string{
 if simsRunning(){return "Sims 4 running. CAS automation is not implemented."}
 return "Sims 4 is not running. Photo importing works offline."
}
func appRoot()string{p,e:=os.Executable();if e!=nil{return "."};return filepath.Dir(p)}
func templatePath()string{return filepath.Join(appRoot(),"Assets","FaceTemplates",map[bool]string{true:"Male",false:"Female"}[gender=="M"],fmt.Sprintf("%s%d.png",gender,selected))}
func libraryStatus()string{libraryMu.RLock();defer libraryMu.RUnlock();if !libraryReady{return "Template library: indexing in background..."};return fmt.Sprintf("Template library: %d/38 images found",libraryCount)}
func scanLibrary(){
 go func(){
  count:=0
  for _,entry:=range []struct{folder,prefix string}{{"Male","M"},{"Female","F"}}{
   dir:=filepath.Join(appRoot(),"Assets","FaceTemplates",entry.folder)
   files,err:=os.ReadDir(dir)
   if err!=nil{logError("Library folder: "+err.Error());continue}
   // One directory read per gender, not 38 individual filesystem lookups.
   found:=make(map[string]bool,len(files))
   for _,f:=range files{if !f.IsDir(){found[strings.ToLower(f.Name())]=true}}
   for i:=1;i<=19;i++{if found[strings.ToLower(fmt.Sprintf("%s%d.png",entry.prefix,i))]{count++}}
  }
  libraryMu.Lock();libraryCount=count;libraryReady=true;libraryMu.Unlock()
  postMessage.Call(mainHWND,WM_LIBRARY_READY,0,0)
 }()
}
func rescanLibrary(){libraryMu.Lock();libraryReady=false;libraryMu.Unlock();change(libraryLabel,libraryStatus());scanLibrary()}
func showTemplate(){p:=templatePath();if _,e:=os.Stat(p);e!=nil{change(statusLabel,"Template image missing: "+p);return};cmd:=exec.Command("rundll32.exe","url.dll,FileProtocolHandler",p);cmd.SysProcAttr=&syscall.SysProcAttr{HideWindow:true,CreationFlags:0x08000000};if e:=cmd.Start();e!=nil{change(statusLabel,"Could not open template preview: "+e.Error())}else{change(statusLabel,"Opened "+fmt.Sprintf("%s%d",gender,selected)+" reference image.")}}
func folders()string{home,_:=os.UserHomeDir();base:=filepath.Join(home,"Documents","Electronic Arts","The Sims 4");if _,e:=os.Stat(base);e==nil{return base};return "The Sims 4 folder not found (configure OneDrive/Documents if needed)"}
func runFaceAnalysis(){
 if photo=="" {change(statusLabel,"Import a reference photo first.");return}
 model:=filepath.Join(appRoot(),"Models","face_landmarker.task")
 info,err:=os.Stat(model)
 if err!=nil || info.Size()<1024*1024 {change(statusLabel,"Face analysis: Models/face_landmarker.task missing or incomplete.");return}
 if !importBusy.CompareAndSwap(false,true) {change(statusLabel,"Analysis or import already in progress.");return}
 change(statusLabel,"Analyzing facial landmarks (please wait)...")
 go func(){
  defer importBusy.Store(false)
  root:=appRoot()
  script:=filepath.Join(root,"FaceEngine","analyze_faces.py")
  output:=filepath.Join(root,"FaceAnalysis","latest.json")
  cas:=filepath.Join(root,"Captures","current_cas.png")
  runtimeExe:=filepath.Join(root,"FaceEngine","runtime","python.exe")
  if _,e:=os.Stat(runtimeExe);e!=nil {
   postStatus("Face engine not installed. Run INSTALL_FACE_ENGINE.cmd once; see FaceEngine/runtime_install.log")
   logError("Face analysis unavailable: bundled runtime missing")
   return
  }
  args:=[]string{script,photo,"--model",filepath.Join(root,"Models","face_landmarker.task"),"--output",output}
  if _,e:=os.Stat(cas);e==nil {args=append(args,"--cas",cas)}
  ctx,cancel:=context.WithTimeout(context.Background(),90*time.Second);defer cancel()
  cmd:=exec.CommandContext(ctx,runtimeExe,args...)
  cmd.Dir=root
  cmd.SysProcAttr=&syscall.SysProcAttr{HideWindow:true,CreationFlags:0x08000000}
  combined,e:=cmd.CombinedOutput()
  logPath:=filepath.Join(root,"FaceEngine","analysis_error.log")
  if e!=nil {
   detail:=strings.TrimSpace(string(combined))
   if len(detail)>450 {detail=detail[len(detail)-450:]}
   if detail=="" {detail=e.Error()}
   _=os.WriteFile(logPath,[]byte(fmt.Sprintf("%v\n%s",e,combined)),0644)
   msg:="Analysis failed: "+detail
   if ctx.Err()!=nil {msg="Face analysis timed out (90s). See FaceEngine/analysis_error.log"}
   if strings.Contains(detail,"No module named") {msg="MediaPipe dependency missing. Re-run INSTALL_FACE_ENGINE.cmd. See analysis_error.log"}
   postStatus(msg)
   return
  }
  var result struct{Reference struct{FaceCount int `json:"face_count"`} `json:"reference"`; CAS *struct{FaceCount int `json:"face_count"`} `json:"cas"`}
  data,readErr:=os.ReadFile(output)
  if readErr!=nil || json.Unmarshal(data,&result)!=nil {postStatus("Analysis ran, but results file is invalid. See FaceAnalysis/latest.json");return}
  if result.Reference.FaceCount==0 {postStatus("No face detected in reference. Use a clear front-facing portrait.");return}
  if result.CAS==nil {postStatus("Reference analyzed successfully. Capture Current Face to compare against CAS.");return}
  if result.CAS.FaceCount==0 {postStatus("Reference analyzed; no CAS face detected. Zoom into the Sim's face, capture again, retry.");return}
  postStatus("SUCCESS: Reference and CAS faces analyzed. See FaceAnalysis/latest.json for landmarks and region measurements.")
 }()
}
func analyze(){if photo==""{change(statusLabel,"Import a photo first.");return};loadPhoto(photo)}
func updateGuide(w,h int){values=[10]int{50,50,50,50,50,50,50,50,50,50};ratio:=float64(w)/float64(h);values[0]=clamp(int(math.Round(50+(ratio-0.75)*24)));values[1]=clamp(100-values[0]);for i,c:=range controls{change(c,labels[i]+": model required")};change(statusLabel,"Image verified. Accurate landmarks require a trained model (not bundled).")}
func clamp(v int)int{if v<0{return 0};if v>100{return 100};return v}
func export(){path:=filepath.Join(appRoot(),"SimMorph_CAS_Guide.json");if path==""{return};obj:=map[string]interface{}{"version":"1.1 stability build","photo":photo,"template":fmt.Sprintf("%s%d",gender,selected),"feature_values":map[string]int{},"note":"No automatic landmarks or CAS sculpting. Face templates are identifiers only."};m:=obj["feature_values"].(map[string]int);for i,k:=range labels{m[k]=values[i]};data,_:=json.MarshalIndent(obj,"","  ");if e:=os.WriteFile(path,data,0644);e!=nil{change(statusLabel,"Export failed: "+e.Error())}else{change(statusLabel,"Guide exported to "+path)}}
var galleryHWND uintptr
var galleryBmp uintptr
var galleryGender = "F"
func openGallery(g string){
 galleryGender=g
 if galleryHWND!=0 {destroyWindow.Call(galleryHWND);galleryHWND=0}
 galleryHWND=win("SimMorphStudioNative","Choose Face Template — click a face to select",WS_OVERLAPPEDWINDOW|WS_VISIBLE,110,70,1020,770,0,0)
 path:=filepath.Join(appRoot(),"Assets",galleryGender+"_gallery.bmp")
 u:=ptr(path);galleryBmp,_,_=loadImage.Call(0,uintptr(unsafe.Pointer(u)),0,0,0,0x10);runtime.KeepAlive(u)
 invalidate.Call(galleryHWND,0,0)
}
func paintGallery(hwnd uintptr){
 var ps paintstruct;dc,_,_:=beginPaint.Call(hwnd,uintptr(unsafe.Pointer(&ps)))
 if galleryBmp!=0 {mem,_,_:=createDC.Call(dc);old,_,_:=selectObj.Call(mem,galleryBmp);stretch.Call(dc,0,0,1000,720,mem,0,0,1000,720,SRCCOPY);selectObj.Call(mem,old);deleteDC.Call(mem)}
 endPaint.Call(hwnd,uintptr(unsafe.Pointer(&ps)))
}
func paint(hwnd uintptr){
 var ps paintstruct;dc,_,_:=beginPaint.Call(hwnd,uintptr(unsafe.Pointer(&ps)))
 previewMu.Lock();background:=bgBmp;previewMu.Unlock();if background!=0 {mem,_,_:=createDC.Call(dc);old,_,_:=selectObj.Call(mem,background);stretch.Call(dc,0,0,1440,900,mem,0,0,1280,720,SRCCOPY);selectObj.Call(mem,old);deleteDC.Call(mem)}
 previewMu.Lock();cas,ref:=previewBmp,referenceBmp;previewMu.Unlock()
 if cas!=0 {mem,_,_:=createDC.Call(dc);old,_,_:=selectObj.Call(mem,cas);stretch.Call(dc,390,115,1000,680,mem,0,0,900,650,SRCCOPY);selectObj.Call(mem,old);deleteDC.Call(mem)}
 if ref!=0 {mem,_,_:=createDC.Call(dc);old,_,_:=selectObj.Call(mem,ref);stretch.Call(dc,25,135,315,365,mem,0,0,900,650,SRCCOPY);selectObj.Call(mem,old);deleteDC.Call(mem)}
 endPaint.Call(hwnd,uintptr(unsafe.Pointer(&ps)))
}
// Native Windows menu: File > About, Quit.
var removeMenu=user32.NewProc("RemoveMenu")
var createMenu=user32.NewProc("CreateMenu")
var createPopupMenu=user32.NewProc("CreatePopupMenu")
var appendMenu=user32.NewProc("AppendMenuW")
var setMenu=user32.NewProc("SetMenu")
var drawMenuBar=user32.NewProc("DrawMenuBar")
var messageBox=user32.NewProc("MessageBoxW")
var destroyWindow=user32.NewProc("DestroyWindow")
const menuAbout=2001
const menuQuit=2002
func populateThemeMenu(){
 if themeMenu==0{return}
 // The entire submenu is recreated when backgrounds are refreshed.
 names:=themeList();themeNames=names
 for i,n:=range names{
  text:=n;if n==activeTheme{text="✓  "+n}
  label:=ptr(text);appendMenu.Call(themeMenu,0,uintptr(themeStart+i),uintptr(unsafe.Pointer(label)))
 }
 appendMenu.Call(themeMenu,0x800,0,0)
 imp:=ptr("Import Background...");appendMenu.Call(themeMenu,0,themeImport,uintptr(unsafe.Pointer(imp)))
 refresh:=ptr("Refresh Background List");appendMenu.Call(themeMenu,0,themeRefresh,uintptr(unsafe.Pointer(refresh)))
}
func rebuildThemeMenu(){
 // Menu items are removed from the end to keep their IDs stable.
 if themeMenu==0{return}
 for {v,_,_:=user32.NewProc("GetMenuItemCount").Call(themeMenu);if int32(v)<=0{break};removeMenu.Call(themeMenu,v-1,0x400)}
 populateThemeMenu();drawMenuBar.Call(mainHWND)
}
func selectTheme(name string){
 activeTheme=name
 _=os.MkdirAll(filepath.Join(appRoot(),"Config"),0755)
 _=os.WriteFile(filepath.Join(appRoot(),"Config","selected_background.txt"),[]byte(name),0644)
 rebuildThemeMenu()
 if strings.ToLower(filepath.Ext(themePath(name)))==".bmp"{loadBackgroundOnUI();change(statusLabel,"Background: "+name);return}
 postStatus("Loading background: "+name)
 go func(){path,err:=saveBMPPreview(themePath(name));if err!=nil{postStatus("Background failed: "+err.Error());return};previewPathMu.Lock();pendingBackgroundPath=path;previewPathMu.Unlock();postMessage.Call(mainHWND,0x800B,0,0)}()
}
var pendingBackgroundPath string
func loadConvertedBackground(){
 previewPathMu.Lock();path:=pendingBackgroundPath;pendingBackgroundPath="";previewPathMu.Unlock()
 if path==""{return}
 u:=ptr(path);bmp,_,_:=loadImage.Call(0,uintptr(unsafe.Pointer(u)),0,0,0,0x10);runtime.KeepAlive(u)
 if bmp==0{postStatus("Unable to load selected background.");return}
 old:=bgBmp;bgBmp=bmp;if old!=0{deleteObj.Call(old)};invalidate.Call(mainHWND,0,0);postStatus("Background: "+activeTheme)
}
func importBackground(){
 path:=fileDialog(false);if path==""{return}
 _=os.MkdirAll(themesDir(),0755)
 base:=filepath.Base(path);dest:=filepath.Join(themesDir(),base)
 if strings.EqualFold(path,dest){selectTheme(base);return}
 in,e:=os.Open(path);if e!=nil{change(statusLabel,e.Error());return};defer in.Close()
 out,e:=os.Create(dest);if e!=nil{change(statusLabel,e.Error());return};_,e=io.Copy(out,in);closeErr:=out.Close();if e==nil{e=closeErr};if e!=nil{change(statusLabel,"Import failed: "+e.Error());return}
 selectTheme(base)
}
// Body region workspace is a planning/calibration aid; it does not infer body measurements or sculpt CAS.
var bodyRegions=[]string{"Neck and shoulders","Upper arms","Forearms and hands","Chest and upper torso","Waist and abdomen","Hips and pelvis","Thighs","Calves and ankles","Feet","Overall proportions"}
func showBodyRegion(index int){
 if index<0||index>=len(bodyRegions){return}
 region:=bodyRegions[index]
 instructions:=map[int]string{0:"Zoom to upper body and inspect neck/shoulder transition.",1:"Use front and side views for upper-arm width.",2:"Inspect forearms and hands in CAS.",3:"Inspect chest and torso from front and side.",4:"Inspect waist contour and abdomen.",5:"Inspect hip width and pelvis silhouette.",6:"Inspect thigh width from front and side.",7:"Inspect calf profile and ankle width.",8:"Inspect feet in full-body view.",9:"Compare overall silhouette and body proportions."}
 text:=region+"\n\n"+instructions[index]+"\n\nBody changes are NOT applied automatically. In Sims 4 CAS, manually select the region and use the game's supported click-and-drag controls. Save a before screenshot first."
 title:=ptr("Body Morph Guide — "+region);msg:=ptr(text);messageBox.Call(mainHWND,uintptr(unsafe.Pointer(msg)),uintptr(unsafe.Pointer(title)),0x40)
 postStatus("Body guide: "+region+" — manual CAS adjustment only.")
}
func exportBodyPlan(){
 data,_:=json.MarshalIndent(map[string]interface{}{"mode":"body_workspace","regions":bodyRegions,"photo":photo,"limitations":"No automatic body landmarks, body-to-CAS mapping or in-game body sculpting implemented.","workflow":"Capture a full-body CAS view, select a region in Body Morph menu, adjust manually in CAS, recapture and compare."},"","  ")
 path:=filepath.Join(appRoot(),"SimMorph_Body_Workspace.json")
 if err:=os.WriteFile(path,data,0644);err!=nil{postStatus("Body plan export failed: "+err.Error())}else{postStatus("Saved body workspace plan: "+path)}
}

// Complete CAS Studio: guided catalog and exportable session plan.
// These guides do not pretend to identify or apply game catalog items automatically.
var casStudioCategories=[]string{
 "Face and skin","Eyes and eyebrows","Nose, cheeks and jaw","Mouth and teeth","Glasses and accessories",
 "Hair and facial hair","Makeup and skin details","Neck and body","Tops and bottoms",
 "Full-body outfits","Shoes and socks","Hats and jewelry","Outfit categories and colors",
}
var casStudioGuides=[]string{
 "Use a front-facing reference and a CAS close-up. Match the head preset and skin details first.",
 "Compare eye spacing, angle, lids and brows. Use Detail Edit Mode for fine sculpting.",
 "Compare nose bridge, tip, cheek fullness, jaw and chin in front and three-quarter views.",
 "Choose a mouth and teeth preset in CAS. Braces require a matching available teeth option or CC.",
 "Inspect the frame shape, lens size and color. Select glasses and other accessories from the installed CAS catalog.",
 "Match cut, length, parting and hair color. Hairstyle selection is manual until catalog matching is built.",
 "Match visible freckles, eyeliner, eyeshadow, lipstick and skin details using CAS choices.",
 "Compare neck and body silhouette from front and side; use CAS-supported drags only.",
 "Match garment type and color; check installed packs and custom content.",
 "Choose a full-body outfit from the available game catalog and match the visible style.",
 "Match shoes, socks and legwear to the reference.",
 "Choose hats, earrings, necklaces, piercings and jewelry in CAS.",
 "Create Everyday, Formal, Athletic, Sleep, Party, Swimwear, Hot Weather and Cold Weather looks separately.",
}
func showCASStudioGuide(index int){
 if index<0||index>=len(casStudioCategories){return}
 body:=casStudioCategories[index]+"\n\n"+casStudioGuides[index]+"\n\nWORKFLOW:\n1. Import reference photo.\n2. Capture the matching CAS view.\n3. Choose the relevant Sims 4 CAS category.\n4. Make edits in game and recapture.\n\nAutomatic item recognition, outfit selection and full-body sculpting are NOT implemented."
 title:=ptr("Complete CAS Studio — "+casStudioCategories[index]);msg:=ptr(body)
 messageBox.Call(mainHWND,uintptr(unsafe.Pointer(msg)),uintptr(unsafe.Pointer(title)),0x40)
 postStatus("CAS Studio guide: "+casStudioCategories[index])
}
func exportCASStudioPlan(){
 plan:=map[string]interface{}{"version":"4.4","mode":"complete_cas_studio","reference_photo":photo,
 "categories":casStudioCategories,"guides":casStudioGuides,
 "workflow":[]string{"Import reference","Capture current CAS view","Choose category","Edit in CAS","Recapture and compare"},
 "limitations":"Guided manual CAS workflow only. No automatic clothing catalog matching, teeth selection, full-body morphing or photo-driven sculpting."}
 data,_:=json.MarshalIndent(plan,"","  ")
 path:=filepath.Join(appRoot(),"SimMorph_Complete_CAS_Plan.json")
 if e:=os.WriteFile(path,data,0644);e!=nil{postStatus("CAS plan export failed: "+e.Error())}else{postStatus("Saved CAS Studio plan: "+path)}
}

func installMenu(hwnd uintptr){
 bar,_,_:=createMenu.Call()
 file,_,_:=createPopupMenu.Call()
 open:=ptr("Open Tabbed Workspace");appendMenu.Call(file,0,4506,uintptr(unsafe.Pointer(open)))
 check:=ptr("Check for Updates");appendMenu.Call(file,0,4510,uintptr(unsafe.Pointer(check)))
 diag:=ptr("Run Diagnostics");appendMenu.Call(file,0,4511,uintptr(unsafe.Pointer(diag)))
 about:=ptr("About");appendMenu.Call(file,0,menuAbout,uintptr(unsafe.Pointer(about)))
 quit:=ptr("Quit");appendMenu.Call(file,0,menuQuit,uintptr(unsafe.Pointer(quit)))
 title:=ptr("File");appendMenu.Call(bar,0x10,file,uintptr(unsafe.Pointer(title)))
 setMenu.Call(hwnd,bar);drawMenuBar.Call(hwnd)
}

func showAbout(hwnd uintptr){
 message:=ptr("Simscraft Studio v5.8\n\nThe Sims 4 CAS companion\n\nImport JPG, JPEG and PNG photos; browse 38 face templates; capture the current CAS view; calibrate and apply face presets.\n\nBody Morph menu provides region guides and plan export. Automatic face/body matching and sculpting are not implemented yet.")
 title:=ptr("About Simscraft Studio")
 messageBox.Call(hwnd,uintptr(unsafe.Pointer(message)),uintptr(unsafe.Pointer(title)),0x40)
}
func wndproc(hwnd uintptr,m uint32,w,l uintptr)uintptr{
 if hwnd==galleryHWND && galleryHWND!=0 {
  switch m {
  case WM_ERASEBKGND:return 1
  case WM_PAINT:paintGallery(hwnd);return 0
  case 0x0201:
   x:=int(int16(uint16(l&0xffff)));y:=int(int16(uint16((l>>16)&0xffff)))
   if x>=0&&x<1000&&y>=0&&y<720 {n:=(y/180)*5+x/200+1;if n<=19 {gender=galleryGender;selected=n;change(templateLabel,fmt.Sprintf("Selected: %s%d",gender,selected));change(autoTemplateLabel,fmt.Sprintf("Manual selection: %s%d",gender,selected));postStatus(fmt.Sprintf("Selected template %s%d. This does not change the in-game Sim.",gender,selected));destroyWindow.Call(hwnd);return 0}}
   return 0
  case WM_DESTROY:galleryHWND=0;if galleryBmp!=0{deleteObj.Call(galleryBmp);galleryBmp=0};return 0
  }
 }
 switch m{case WM_ERASEBKGND:return 1;case WM_PAINT:paint(hwnd);return 0;case WM_PHOTO_READY:finishPhoto();return 0;case WM_DIALOG_READY:finishBrowse();return 0;case WM_LIBRARY_READY:change(libraryLabel,libraryStatus());return 0;case 0x8007:invalidate.Call(hwnd,0,0);return 0;case 0x8008:loadPreviewOnUI(true);return 0;case 0x8009:loadPreviewOnUI(false);return 0;case 0x800A:selectTheme(activeTheme);return 0;case 0x800B:loadConvertedBackground();return 0;case 0x8006:gameMu.Lock();v:=gameText;gameMu.Unlock();if gameLabel!=0{change(gameLabel,v)};return 0;case 0x0312:applyCAS();return 0;case 0x8004:change(statusLabel,fmt.Sprintf("CAS position captured: (%d,%d). Switch to Sims 4 to apply.",casX,casY));return 0;case 0x8005:statusMu.Lock();v:=pendingStatus;statusMu.Unlock();change(statusLabel,v);return 0;case WM_COMMAND:id:=int(w&0xffff);if id>=4300&&id<4300+len(casStudioCategories){showCASStudioGuide(id-4300);return 0};if id==4399{exportCASStudioPlan();return 0};if id==4506{openStudioTabs();return 0};if id==4510{runUpdateCheck();return 0};if id==4511{runDiagnostics();return 0};if id==4501{openGlassesCreator();return 0};if id==4502{openSmartAssist();return 0};if id==4503{openSmartMorph();return 0};if id==4504{openFaceTextureLab();return 0};if id==4505{openUVFaceAlign();return 0};if id>=4100&&id<4100+len(bodyRegions){showBodyRegion(id-4100);return 0};if id==4199{exportBodyPlan();return 0};if id==3010{openGallery("F");return 0};if id==3011{openGallery("M");return 0};if id>=themeStart&&id<themeStart+len(themeNames){selectTheme(themeNames[id-themeStart]);return 0};switch id{case themeImport:importBackground();case themeRefresh:rebuildThemeMenu();change(statusLabel,"Background list refreshed.");case menuAbout:showAbout(hwnd);case menuQuit:destroyWindow.Call(hwnd);case 101:browsePhoto();case 102:gender="M";change(templateLabel,fmt.Sprintf("Selected: M%d",selected));case 103:gender="F";change(templateLabel,fmt.Sprintf("Selected: F%d",selected));case 104:if selected>1{selected--};change(templateLabel,fmt.Sprintf("Selected: %s%d",gender,selected));case 105:if selected<19{selected++};change(templateLabel,fmt.Sprintf("Selected: %s%d",gender,selected));case 106:analyze();case 107:export();case 108:change(folderLabel,folders());case 110:showTemplate();case 111:rescanLibrary();case 112:calibrateCAS();case 113:applyCAS();case 114:openModels();case 115:refreshModel();case 116:captureCurrentCAS();case 117:captureGridPoint(true);case 118:captureGridPoint(false);case 119:applySelectedPreset();case 120:runFaceAnalysis();case 121:refreshPreview();case 122:calibrateCAS();case 123:applyCAS()};return 0;case WM_DESTROY:user32.NewProc("PostQuitMessage").Call(0);return 0};v,_,_:=defProc.Call(hwnd,uintptr(m),w,l);return v}
func main(){
 if exe,e:=os.Executable();e==nil{updater:=filepath.Join(filepath.Dir(exe),"SimcraftUpdater.exe");if _,e=os.Stat(updater);e==nil{cmd:=exec.Command(updater,"check");if e=cmd.Run();e!=nil{if exitErr,ok:=e.(*exec.ExitError);ok&&exitErr.ExitCode()==42{return}}}}

 // Win32 window creation and its message pump MUST stay on one OS thread.
 runtime.LockOSThread()
 defer runtime.UnlockOSThread()
 logError("v3.4 startup: locked Windows UI thread")
 h,_,_:=getModule.Call(0)
 name:=ptr("SimMorphStudioNative")
 iconFile:=filepath.Join(filepath.Dir(os.Args[0]),"Simcraft-Icon.ico")
 iconHandle,_,_:=loadImage.Call(0,uintptr(unsafe.Pointer(ptr(iconFile))),1,0,0,0x10)
 c:=wc{Size:uint32(unsafe.Sizeof(wc{})),WndProc:syscall.NewCallback(wndproc),Instance:h,ClassName:name,Background:0,Icon:iconHandle,IconSm:iconHandle}
 regClass.Call(uintptr(unsafe.Pointer(&c)))
 activeTheme=readThemeSelection()
 valid:=false;for _,n:=range themeList(){if n==activeTheme{valid=true;break}};if !valid{activeTheme="Default CAS Room"}
 mainHWND=win("SimMorphStudioNative","Simscraft Studio — CAS Workspace",WS_OVERLAPPEDWINDOW|WS_VISIBLE,45,20,1456,938,0,0)
 installMenu(mainHWND)
 button("Open Simcraft Tabs",25,47,315,38,4506)
 // SAFE STARTUP: no background GDI loading or concurrent process checks.
 // This isolates the immediate startup freeze; preview and photo controls remain.
 label("SIMSCRAFT STUDIO  |  CAS WORKSPACE",25,15,600,28)
 gameLabel=label("Safe startup mode — automatic detection paused",910,15,490,28)
 label("REFERENCE PHOTO",25,93,300,28)
 button("Browse for Photo",25,520,315,38,101)
 autoTemplateLabel=label("Auto template suggestion runs after importing a photo",25,605,335,25)
 button("Female Templates (F1–F19)",25,645,315,36,3010)
 button("Male Templates (M1–M19)",25,690,315,36,3011)
 photoLabel=label("No photo selected",25,565,330,34)
 label("CURRENT SIM  |  CAS PREVIEW",390,82,560,30)
 button("Capture Current CAS Face",390,810,300,40,116)
 button("Refresh Preview",710,810,205,40,121)
 button("Analyze Faces",935,810,200,40,120)
 button("Set CAS Click Point",390,755,255,38,122)
 button("Apply One CAS Click",665,755,255,38,123)
 label("Manual calibrated click only — NOT automatic face morphing",935,755,440,38)
 statusLabel=label("Ready. Import a photo and capture the current Sim in CAS.",25,860,1350,30)
 // Retain these for existing asynchronous update handlers; advanced controls are not displayed.
 templateLabel=label("Selected: F1 (default)",25,733,335,28);libraryLabel=label("",0,0,0,0)
 folderLabel=label("",0,0,0,0);modelLabel=label("",0,0,0,0)
 for i:=range controls{controls[i]=label("",0,0,0,0)}
 showWin.Call(mainHWND,5);updateWin.Call(mainHWND)
 // Restore background on the UI thread after window creation.
 postMessage.Call(mainHWND,0x800A,0,0)
 // Keep automatic game scanning and template indexing paused for stability.
 var m msg
 for{r,_,_:=getMsg.Call(uintptr(unsafe.Pointer(&m)),0,0,0);if int32(r)<=0{break};transMsg.Call(uintptr(unsafe.Pointer(&m)));dispatch.Call(uintptr(unsafe.Pointer(&m)))}
 if bgBmp!=0{deleteObj.Call(bgBmp)};if previewBmp!=0{deleteObj.Call(previewBmp)};if referenceBmp!=0{deleteObj.Call(referenceBmp)}
}

func openGlassesCreator(){ exe,err:=os.Executable();if err!=nil{return};path:=filepath.Join(filepath.Dir(exe),"GlassesCreator.html");if _,err:=os.Stat(path);err!=nil{change(statusLabel,"Glasses Creator file missing beside executable");return};cmd:=exec.Command("rundll32.exe","url.dll,FileProtocolHandler",path);cmd.SysProcAttr=&syscall.SysProcAttr{HideWindow:true,CreationFlags:0x08000000};if err:=cmd.Start();err!=nil{change(statusLabel,"Cannot open Glasses Creator: "+err.Error())}else{change(statusLabel,"Glasses Creator prototype opened in browser")}}

func openSmartAssist(){exe,err:=os.Executable();if err!=nil{return};path:=filepath.Join(filepath.Dir(exe),"SmartAssist.html");if _,err:=os.Stat(path);err!=nil{change(statusLabel,"SmartAssist.html missing beside executable");return};cmd:=exec.Command("rundll32.exe","url.dll,FileProtocolHandler",path);cmd.SysProcAttr=&syscall.SysProcAttr{HideWindow:true,CreationFlags:0x08000000};if err:=cmd.Start();err!=nil{change(statusLabel,"Cannot open SmartAssist: "+err.Error())}else{change(statusLabel,"SmartAssist and CAS Mod Studio opened in browser")}}

func openSmartMorph(){exe,err:=os.Executable();if err!=nil{return};path:=filepath.Join(filepath.Dir(exe),"SmartMorph.html");if _,err:=os.Stat(path);err!=nil{change(statusLabel,"SmartMorph.html missing beside executable");return};cmd:=exec.Command("rundll32.exe","url.dll,FileProtocolHandler",path);cmd.SysProcAttr=&syscall.SysProcAttr{HideWindow:true,CreationFlags:0x08000000};if err:=cmd.Start();err!=nil{change(statusLabel,"Cannot open SmartMorph: "+err.Error())}else{change(statusLabel,"SmartMorph photo workspace opened in browser")}}

func openFaceTextureLab(){exe,err:=os.Executable();if err!=nil{return};path:=filepath.Join(filepath.Dir(exe),"FaceTextureLab.html");if _,err:=os.Stat(path);err!=nil{change(statusLabel,"FaceTextureLab.html missing beside executable");return};cmd:=exec.Command("rundll32.exe","url.dll,FileProtocolHandler",path);cmd.SysProcAttr=&syscall.SysProcAttr{HideWindow:true,CreationFlags:0x08000000};if err:=cmd.Start();err!=nil{change(statusLabel,"Cannot open Face Texture Lab: "+err.Error())}else{change(statusLabel,"Face Texture Lab opened in browser")}}

func openUVFaceAlign(){exe,err:=os.Executable();if err!=nil{return};path:=filepath.Join(filepath.Dir(exe),"UVFaceAlign.html");if _,err:=os.Stat(path);err!=nil{change(statusLabel,"UVFaceAlign.html missing beside executable");return};cmd:=exec.Command("rundll32.exe","url.dll,FileProtocolHandler",path);cmd.SysProcAttr=&syscall.SysProcAttr{HideWindow:true,CreationFlags:0x08000000};if err:=cmd.Start();err!=nil{change(statusLabel,"Cannot open UV Alignment: "+err.Error())}else{change(statusLabel,"UV Face Alignment opened")}}

func openStudioTabs(){exe,err:=os.Executable();if err!=nil{return};path:=filepath.Join(filepath.Dir(exe),"Studio.html");if _,err:=os.Stat(path);err!=nil{change(statusLabel,"Studio.html missing beside executable");return};cmd:=exec.Command("rundll32.exe","url.dll,FileProtocolHandler",path);cmd.SysProcAttr=&syscall.SysProcAttr{HideWindow:true,CreationFlags:0x08000000};if err:=cmd.Start();err!=nil{change(statusLabel,"Cannot open studio: "+err.Error())}else{change(statusLabel,"Simscraft Studio tabs opened")}}

func runUpdateCheck(){
 exe,err:=os.Executable();if err!=nil{return}
 helper:=filepath.Join(filepath.Dir(exe),"SimcraftUpdater.exe")
 cmd:=exec.Command(helper,"check")
 cmd.SysProcAttr=&syscall.SysProcAttr{HideWindow:true,CreationFlags:0x08000000}
 go func(){err:=cmd.Run();if err!=nil{postStatus("Update check: "+err.Error())}else{postStatus("Update check complete. See Diagnostics for configuration.")}}()
}
func runDiagnostics(){
 exe,err:=os.Executable();if err!=nil{return}
 helper:=filepath.Join(filepath.Dir(exe),"SimcraftUpdater.exe")
 cmd:=exec.Command(helper,"diagnose")
 cmd.SysProcAttr=&syscall.SysProcAttr{HideWindow:true,CreationFlags:0x08000000}
 output,err:=cmd.CombinedOutput()
 report:=filepath.Join(filepath.Dir(exe),"Simcraft-Diagnostics.txt")
 if err!=nil{output=append(output,[]byte("\\nDiagnostic error: "+err.Error())...)}
 if e:=os.WriteFile(report,output,0644);e!=nil{postStatus("Could not save diagnostics: "+e.Error());return}
 launch:=exec.Command("notepad.exe",report);_ =launch.Start()
 postStatus("Diagnostic report saved: "+report)
}
