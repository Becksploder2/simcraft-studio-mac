package main

import (
 "archive/zip"
 "bytes"
 urlpkg "net/url"
 "crypto/ed25519"
 "crypto/sha256"
 "encoding/base64"
 "encoding/hex"
 "encoding/json"
 "errors"
 "fmt"
 "io"
 "net/http"
 "os"
 "os/exec"
 "path/filepath"
 "runtime"
 "strconv"
 "strings"
 "syscall"
 "unsafe"
 "time"
)

const version="5.8.0"
var manifestURL="" // Optional direct HTTPS signed manifest override
var githubOwner="Becksploder2" // Set at build time with -ldflags
var githubRepo="Simcraft" // Set at build time with -ldflags
var publicKey="" // Base64 Ed25519 PUBLIC key; set by publisher at build time
type Artifact struct { URL string `json:"url"`; SHA256 string `json:"sha256"`; Size int64 `json:"size"` }
type Manifest struct { App string `json:"app"`; Version string `json:"version"`; Notes string `json:"notes"`; Windows Artifact `json:"windows"`; Mac Artifact `json:"mac"` }
type Signed struct { Payload json.RawMessage `json:"payload"`; Signature string `json:"signature"` }
func client()*http.Client{return &http.Client{Timeout:30*time.Second,CheckRedirect:func(req *http.Request,via []*http.Request)error{if len(via)>4||req.URL.Scheme!="https"{return errors.New("insecure redirect")};return nil}}}
func httpsURL(u string)bool{return strings.HasPrefix(u,"https://") && !strings.ContainsAny(u,"\r\n")}
func fetchManifest(url,key string)(Manifest,error){
 var m Manifest
 if !httpsURL(url){return m,errors.New("manifest must use HTTPS")}
 pub,e:=base64.StdEncoding.DecodeString(key);if e!=nil||len(pub)!=ed25519.PublicKeySize{return m,errors.New("publisher public key not configured")}
 r,e:=client().Get(url);if e!=nil{return m,e};defer r.Body.Close()
 if r.StatusCode!=200{return m,fmt.Errorf("manifest HTTP %d",r.StatusCode)}
 raw,e:=io.ReadAll(io.LimitReader(r.Body,128*1024+1));if e!=nil||len(raw)>128*1024{return m,errors.New("manifest too large or unreadable")}
 var s Signed;if e=json.Unmarshal(raw,&s);e!=nil{return m,e}
 sig,e:=base64.StdEncoding.DecodeString(s.Signature);if e!=nil||!ed25519.Verify(pub,s.Payload,sig){return m,errors.New("release signature verification failed")}
 if e=json.Unmarshal(s.Payload,&m);e!=nil{return m,e}
 if m.App!="simcraft"||!validVersion(m.Version){return m,errors.New("invalid app/version")}
 return m,nil
}
func validVersion(v string)bool{parts:=strings.Split(v,".");if len(parts)!=3{return false};for _,x:=range parts{if x==""||len(x)>6{return false};for _,c:=range x{if c<'0'||c>'9'{return false}}};return true}
func newer(a,b string)bool{aa:=strings.Split(a,".");bb:=strings.Split(b,".");for i:=0;i<3;i++{x,_:=strconv.Atoi(aa[i]);y,_:=strconv.Atoi(bb[i]);if x!=y{return x>y}};return false}
func artifact(m Manifest)Artifact{if runtime.GOOS=="darwin"{return m.Mac};return m.Windows}
func download(a Artifact,dest string)error{
 if !httpsURL(a.URL)||len(a.SHA256)!=64||a.Size<1||a.Size>500*1024*1024{return errors.New("invalid release artifact")}
 r,e:=client().Get(a.URL);if e!=nil{return e};defer r.Body.Close()
 if r.StatusCode!=200{return fmt.Errorf("download HTTP %d",r.StatusCode)}
 f,e:=os.Create(dest);if e!=nil{return e};defer f.Close()
 h:=sha256.New();n,e:=io.Copy(io.MultiWriter(f,h),io.LimitReader(r.Body,a.Size+1));if e!=nil{return e}
 if n!=a.Size||!strings.EqualFold(hex.EncodeToString(h.Sum(nil)),a.SHA256){return errors.New("download size or SHA-256 mismatch")}
 return f.Sync()
}
func extract(zippath,dest string)error{
 zr,e:=zip.OpenReader(zippath);if e!=nil{return e};defer zr.Close()
 if len(zr.File)>5000{return errors.New("too many archive entries")}
 for _,z:=range zr.File{
  name:=filepath.FromSlash(z.Name)
  if filepath.IsAbs(name)||strings.Contains(name,":")||name==".."||strings.HasPrefix(name,".."+string(os.PathSeparator))||strings.Contains(name,string(os.PathSeparator)+".."+string(os.PathSeparator))||z.Mode()&os.ModeSymlink!=0{return errors.New("unsafe archive path")}
  target:=filepath.Join(dest,name);rel,e:=filepath.Rel(dest,target);if e!=nil||rel==".."||strings.HasPrefix(rel,".."+string(os.PathSeparator)){return errors.New("archive escapes target")}
  if z.FileInfo().IsDir(){if e=os.MkdirAll(target,0755);e!=nil{return e};continue}
  if z.UncompressedSize64>300*1024*1024{return errors.New("oversized entry")}
  if e=os.MkdirAll(filepath.Dir(target),0755);e!=nil{return e}
  src,e:=z.Open();if e!=nil{return e}
  dst,e:=os.OpenFile(target,os.O_CREATE|os.O_EXCL|os.O_WRONLY,0644);if e!=nil{src.Close();return e}
  n,copyErr:=io.Copy(dst,io.LimitReader(src,int64(z.UncompressedSize64)+1));closeErr:=dst.Close();src.Close()
  if copyErr!=nil||closeErr!=nil||n!=int64(z.UncompressedSize64){return errors.New("archive entry extraction failed")}
 }
 return nil
}
func prompt(message string)bool{
 if runtime.GOOS!="windows"{fmt.Println(message);fmt.Print("Update now? [y/N] ");var s string;fmt.Scanln(&s);return strings.EqualFold(s,"y")}
 dll:=syscall.NewLazyDLL("user32.dll");proc:=dll.NewProc("MessageBoxW")
 title,_:=syscall.UTF16PtrFromString("Simcraft Update Available")
 msg,_:=syscall.UTF16PtrFromString(message)
 r,_,_:=proc.Call(0,uintptr(unsafe.Pointer(msg)),uintptr(unsafe.Pointer(title)),0x24)
 _=msg
 return r==6
}
type githubRelease struct {
 TagName string `json:"tag_name"`
 Draft bool `json:"draft"`
 Prerelease bool `json:"prerelease"`
 Assets []struct {Name string `json:"name"`; URL string `json:"browser_download_url"`} `json:"assets"`
}
func githubManifestURL(owner,repo string)(string,string,error){
 if !validSlug(owner)||!validSlug(repo){return "","",errors.New("invalid GitHub owner or repository")}
 endpoint:="https://api.github.com/repos/"+owner+"/"+repo+"/releases/latest"
 req,e:=http.NewRequest("GET",endpoint,nil);if e!=nil{return "","",e}
 req.Header.Set("Accept","application/vnd.github+json")
 req.Header.Set("User-Agent","Simcraft-Updater/5.7")
 r,e:=client().Do(req);if e!=nil{return "","",e};defer r.Body.Close()
 if r.StatusCode!=200{return "","",fmt.Errorf("GitHub latest release HTTP %d",r.StatusCode)}
 raw,e:=io.ReadAll(io.LimitReader(r.Body,256*1024+1));if e!=nil||len(raw)>256*1024{return "","",errors.New("GitHub response too large")}
 var release githubRelease;if e=json.Unmarshal(raw,&release);e!=nil{return "","",e}
 if release.Draft||release.Prerelease{return "","",errors.New("no stable release")}
 for _,a:=range release.Assets{
  if a.Name=="simcraft-release.json"{
   u,e:=urlpkg.Parse(a.URL);if e!=nil||u.Scheme!="https"||u.Host!="github.com"||!strings.HasPrefix(u.Path,"/"+owner+"/"+repo+"/releases/download/"){return "","",errors.New("unexpected GitHub manifest URL")}
   return a.URL,release.TagName,nil
  }
 }
 return "","",errors.New("latest GitHub release has no simcraft-release.json asset")
}
func validSlug(s string)bool{
 if len(s)<1||len(s)>100{return false}
 for _,c:=range s{if !(c>='a'&&c<='z'||c>='A'&&c<='Z'||c>='0'&&c<='9'||c=='-'||c=='_'||c=='.'){return false}}
 return s!="."&&s!=".."
}
func waitForPID(pid int,timeout time.Duration)error{
 if runtime.GOOS=="windows"{
  kernel:=syscall.NewLazyDLL("kernel32.dll")
  open:=kernel.NewProc("OpenProcess")
  wait:=kernel.NewProc("WaitForSingleObject")
  closeHandle:=kernel.NewProc("CloseHandle")
  h,_,err:=open.Call(0x00100000,0,uintptr(pid)) // SYNCHRONIZE
  if h==0{return fmt.Errorf("cannot open running application process: %v",err)}
  defer closeHandle.Call(h)
  r,_,_:=wait.Call(h,uintptr(timeout.Milliseconds()))
  if r!=0{return errors.New("application did not exit; update canceled")}
  return nil
 }
 p,e:=os.FindProcess(pid);if e!=nil{return nil}
 for start:=time.Now();time.Since(start)<timeout;time.Sleep(200*time.Millisecond){if e=p.Signal(syscall.Signal(0));e!=nil{return nil}}
 return errors.New("application did not exit; update canceled")
}
func runCheck()error{
 if publicKey==""{return nil} // Not configured: do not claim updates are available
 url:=manifestURL
 var releaseTag string
 if githubOwner!=""&&githubRepo!="" {
  var err error
  url,releaseTag,err=githubManifestURL(githubOwner,githubRepo)
  if err!=nil{return err}
 }
 if url==""{return nil}
 m,e:=fetchManifest(url,publicKey);if e!=nil{return e}
 if releaseTag!="" && releaseTag!="v"+m.Version && releaseTag!=m.Version{return errors.New("signed manifest version does not match GitHub release tag") }
 if !newer(m.Version,version){return nil}
 a:=artifact(m)
 if githubOwner!="" && githubRepo!="" {
  u,e:=urlpkg.Parse(a.URL)
  if e!=nil || u.Scheme!="https" || u.Host!="github.com" || !strings.HasPrefix(u.Path,"/"+githubOwner+"/"+githubRepo+"/releases/download/") {return errors.New("signed artifact does not belong to configured repository")}
 }
 if !prompt(fmt.Sprintf("Simcraft %s is available.\n\n%s\n\nDownload, verify, and install?",m.Version,m.Notes)){return nil}
 exe,e:=os.Executable();if e!=nil{return e}
 root:=filepath.Dir(exe)
 stage,e:=os.MkdirTemp("","simcraft-update-*");if e!=nil{return e}
 archive:=filepath.Join(stage,"update.zip")
 if e=download(a,archive);e!=nil{return e}
 extracted:=filepath.Join(stage,"payload");if e=os.MkdirAll(extracted,0700);e!=nil{return e}
 if e=extract(archive,extracted);e!=nil{return e}
 if _,e=os.Stat(filepath.Join(extracted,filepath.Base(exe)));e!=nil{return fmt.Errorf("update missing executable %s",filepath.Base(exe))}
 helper:=filepath.Join(root,"SimcraftUpdater.exe")
 if runtime.GOOS!="windows"{helper=filepath.Join(root,"SimcraftUpdater")}
 // Copy updater outside the install folder so Windows can replace it.
 helperCopy:=filepath.Join(stage,"SimcraftUpdater.exe")
 if runtime.GOOS!="windows"{helperCopy=filepath.Join(stage,"SimcraftUpdater")}
 original,e:=os.Open(helper);if e!=nil{return e};defer original.Close()
 out,e:=os.Create(helperCopy);if e!=nil{return e}
 _,e=io.Copy(out,original);out.Close();if e!=nil{return e}
 cmd:=exec.Command(helperCopy,"install",strconv.Itoa(os.Getppid()),extracted,root,filepath.Base(exe))
 if e=cmd.Start();e!=nil{return e}
 fmt.Println("Verified update staged; exiting so installer can replace files.")
 os.Exit(42)
 return nil
}
func install(pid int,src,root,exeName string)error{
 if !filepath.IsAbs(src)||!filepath.IsAbs(root){return errors.New("invalid paths")}
 // Wait until main program exits; Windows will deny replacement while running.
 if e:=waitForPID(pid,60*time.Second);e!=nil{return e}
 backup:=root+".simcraft-backup"
 if _,e:=os.Stat(backup);e==nil{return errors.New("existing backup needs attention")}
 if e:=os.Rename(root,backup);e!=nil{return fmt.Errorf("cannot back up installation: %w",e)}
 if e:=os.Rename(src,root);e!=nil{_ = os.Rename(backup,root);return fmt.Errorf("install failed, old version restored: %w",e)}
 cmd:=exec.Command(filepath.Join(root,exeName))
 if e:=cmd.Start();e!=nil{
  _=os.Rename(root,root+".simcraft-failed")
  _=os.Rename(backup,root)
  return fmt.Errorf("new app did not launch; rollback attempted: %w",e)
 }
 fmt.Println("Updated. Backup retained at:",backup)
 return nil
}
func diagnostics() {
 fmt.Println("Simcraft v"+version+" diagnostics")
 fmt.Println("GitHub repository:",githubOwner+"/"+githubRepo)
 fmt.Println("Signed updates configured:",publicKey!="")
 fmt.Println("Updater platform:",runtime.GOOS+"/"+runtime.GOARCH)
 exe,e:=os.Executable();if e!=nil{fmt.Println("Executable path:",e);return}
 root:=filepath.Dir(exe)
 fmt.Println("Install directory:",root)
 for _,f:=range []string{"SimcraftStudio.exe","SimcraftUpdater.exe","Studio.html","Simcraft-Icon.ico"}{
  _,err:=os.Stat(filepath.Join(root,f));fmt.Println("File",f,"present:",err==nil)
 }
 test,e:=os.CreateTemp(root,".simcraft-writecheck-*");if e!=nil{fmt.Println("Install folder writable: false",e)}else{fmt.Println("Install folder writable: true");test.Close();os.Remove(test.Name())}
 fmt.Println("No private keys or tokens are required on player machines.")
 if publicKey==""{fmt.Println("Update channel: NOT ACTIVE. Publisher signing key has not been configured.")}
}
func main(){
 if len(os.Args)>1&&os.Args[1]=="diagnose"{diagnostics();return}

 if len(os.Args)>1&&os.Args[1]=="install"{
  if len(os.Args)!=6{fmt.Fprintln(os.Stderr,"usage: install PID staged-folder install-folder executable");os.Exit(2)}
  pid,e:=strconv.Atoi(os.Args[2]);if e!=nil{os.Exit(2)}
  if e=install(pid,os.Args[3],os.Args[4],os.Args[5]);e!=nil{fmt.Fprintln(os.Stderr,e);os.Exit(1)};return
 }
 if e:=runCheck();e!=nil{fmt.Fprintln(os.Stderr,"Update check skipped:",e)}
 _=bytes.MinRead // keep standard-library-only dependencies
}
