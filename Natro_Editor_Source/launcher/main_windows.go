//go:build windows

// Natro Editor launcher — reconstructed source, not original native core source.
// Starts a bundled desktop editor. On first use, looks for a NatroMacro installation,
// puts a verified copy of the editor beside patterns, and makes a Desktop shortcut.
// No extra selection dialog, startup registration, or policy modification.
package main

import (
    "crypto/sha256"
    _ "embed"
    "errors"
    "fmt"
    "os"
    "path/filepath"
    "runtime"
    "strings"
    "syscall"
    "time"
    "unsafe"
)

//go:embed Natro_Editor_core.exe
var coreExe []byte

var (
    shell32 = syscall.NewLazyDLL("shell32.dll")
    ole32 = syscall.NewLazyDLL("ole32.dll")
    shellExecuteW = shell32.NewProc("ShellExecuteW")
    shGetKnownFolderPath = shell32.NewProc("SHGetKnownFolderPath")
    shChangeNotify = shell32.NewProc("SHChangeNotify")
    coTaskMemFree = ole32.NewProc("CoTaskMemFree")
    coInitializeEx = ole32.NewProc("CoInitializeEx")
    coUninitialize = ole32.NewProc("CoUninitialize")
    coCreateInstance = ole32.NewProc("CoCreateInstance")
)

func utf16(s string) *uint16 { p, _ := syscall.UTF16PtrFromString(s); return p }
func logError(msg string) {
    local := os.Getenv("LOCALAPPDATA")
    if local == "" { return }
    d := filepath.Join(local, "HivePathStudio")  // preserve earlier launcher state
    if err := os.MkdirAll(d, 0700); err != nil { return }
    f, err := os.OpenFile(filepath.Join(d,"launcher.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
    if err != nil { return }; defer f.Close()
    _, _ = fmt.Fprintf(f, "%s %s\n", time.Now().Format(time.RFC3339), msg)
}

func installedPathFile() string {
    if local := os.Getenv("LOCALAPPDATA"); local != "" {
        return filepath.Join(local, "HivePathStudio", "installed_target.txt")
    }
    return ""
}
func rootWithPatterns(candidate string) (string,error) {
    candidate = filepath.Clean(candidate)
    if strings.EqualFold(filepath.Base(candidate), "patterns") { candidate=filepath.Dir(candidate) }
    if _, ok := looksLikeNatroRoot(candidate); !ok { return "", errors.New("not an identifiable NatroMacro installation") }
    return candidate,nil
}
func readStoredRoot() string {
    state:=installedPathFile(); if state=="" {return ""}
    data,err:=os.ReadFile(state);if err!=nil{return ""}
    target:=strings.TrimSpace(string(data))
    if !strings.EqualFold(filepath.Base(target),"Natro Editor.exe") &&
       !strings.EqualFold(filepath.Base(target),"HivePath_Studio.exe") {return ""}
    root,err:=rootWithPatterns(filepath.Dir(target));if err!=nil{return ""}
    return root
}
func saveState(target string) error {
    path:=installedPathFile();if path=="" {return errors.New("LOCALAPPDATA unavailable")}
    if err:=os.MkdirAll(filepath.Dir(path),0700);err!=nil{return err}
    tmp:=path+".tmp"
    if err:=os.WriteFile(tmp,[]byte(target+"\r\n"),0600);err!=nil{return err}
    return os.Rename(tmp,path)
}

func isNatroEditorExecutable(path string) bool {
    data,err:=os.ReadFile(path)
    if err!=nil || len(data)<8192 || string(data[:2])!="MZ" {return false}
    return strings.Contains(string(data),"Natro Editor | Bee Swarm Field Route Editor") ||
       strings.Contains(string(data),"HivePath Studio | Bee Swarm Field Route Editor")
}
func verifiedWrite(target string, contents []byte) error {
    desired:=sha256.Sum256(contents)
    current,err:=os.ReadFile(target)
    if err==nil {
        if sha256.Sum256(current)==desired {return nil}
        if !isNatroEditorExecutable(target) {return fmt.Errorf("refusing to replace unrelated executable %s",target)}
    } else if !errors.Is(err,os.ErrNotExist) {return err}
    if err:=os.MkdirAll(filepath.Dir(target),0700);err!=nil{return err}
    f,err:=os.CreateTemp(filepath.Dir(target),".natro-editor-stage-*.exe")
    if err!=nil{return err}
    staged:=f.Name();defer os.Remove(staged)
    if _,err=f.Write(contents);err!=nil{f.Close();return err}
    if err=f.Sync();err!=nil{f.Close();return err}
    if err=f.Close();err!=nil{return err}
    c,err:=os.ReadFile(staged);if err!=nil{return err}
    if sha256.Sum256(c)!=desired {return errors.New("checksum mismatch after staging")}
    if current!=nil {
        backup:=filepath.Join(filepath.Dir(target),"Natro_Editor_backup_"+time.Now().Format("20060102_150405.000")+".exe")
        if err:=os.Rename(target,backup);err!=nil{return err}
        if err:=os.Rename(staged,target);err!=nil{_ = os.Rename(backup,target);return err}
    } else if err:=os.Rename(staged,target);err!=nil{return err}
    return nil
}

// Native Windows ShellLink COM: no external scripting tools required.
type guid struct { Data1 uint32; Data2 uint16; Data3 uint16; Data4 [8]byte }
var (
    clsidShellLink=guid{0x00021401,0,0,[8]byte{0xC0,0,0,0,0,0,0,0x46}}
    iidShellLinkW=guid{0x000214F9,0,0,[8]byte{0xC0,0,0,0,0,0,0,0x46}}
    iidPersistFile=guid{0x0000010B,0,0,[8]byte{0xC0,0,0,0,0,0,0,0x46}}
    folderIDDesktop=guid{0xB4BFCC3A,0xDB2C,0x424C,[8]byte{0xB0,0x29,0x7F,0xE9,0x9A,0x87,0xC6,0x41}}
)
func comCall(obj uintptr,index int,args ...uintptr) uintptr {
    vtbl:=*(*uintptr)(unsafe.Pointer(obj))
    fn:=*(*uintptr)(unsafe.Pointer(vtbl+uintptr(index)*unsafe.Sizeof(uintptr(0))))
    full:=append([]uintptr{obj},args...)
    r,_,_:=syscall.SyscallN(fn,full...)
    return r
}
func hrCheck(hr uintptr,action string) error {
    if int32(uint32(hr))<0 {return fmt.Errorf("%s failed: HRESULT 0x%08x",action,uint32(hr))}
    return nil
}
func shellLink() (uintptr,func(),error) {
    runtime.LockOSThread()
    hr,_,_:=coInitializeEx.Call(0,2)
    if err:=hrCheck(hr,"CoInitializeEx");err!=nil{runtime.UnlockOSThread();return 0,nil,err}
    var ptr uintptr
    hr,_,_=coCreateInstance.Call(uintptr(unsafe.Pointer(&clsidShellLink)),0,1,
        uintptr(unsafe.Pointer(&iidShellLinkW)),uintptr(unsafe.Pointer(&ptr)))
    if err:=hrCheck(hr,"CoCreateInstance");err!=nil{coUninitialize.Call();runtime.UnlockOSThread();return 0,nil,err}
    return ptr,func(){comCall(ptr,2);coUninitialize.Call();runtime.UnlockOSThread()},nil
}
func persistFile(link uintptr) (uintptr,error) {
    var persist uintptr
    if err:=hrCheck(comCall(link,0,uintptr(unsafe.Pointer(&iidPersistFile)),uintptr(unsafe.Pointer(&persist))),"QueryInterface(IPersistFile)");err!=nil{return 0,err}
    return persist,nil
}
func readShortcutTarget(path string) (string,error) {
    link,release,err:=shellLink();if err!=nil{return "",err};defer release()
    p,err:=persistFile(link);if err!=nil{return "",err};defer comCall(p,2)
    if err:=hrCheck(comCall(p,5,uintptr(unsafe.Pointer(utf16(path))),0),"Load");err!=nil{return "",err}
    var buf [32768]uint16
    if err:=hrCheck(comCall(link,3,uintptr(unsafe.Pointer(&buf[0])),uintptr(len(buf)),0,0),"GetPath");err!=nil{return "",err}
    return syscall.UTF16ToString(buf[:]),nil
}
func writeShortcut(linkPath,target string) error {
    link,release,err:=shellLink();if err!=nil{return err};defer release()
    if err:=hrCheck(comCall(link,20,uintptr(unsafe.Pointer(utf16(target)))),"SetPath");err!=nil{return err}
    if err:=hrCheck(comCall(link,9,uintptr(unsafe.Pointer(utf16(filepath.Dir(target))))),"SetWorkingDirectory");err!=nil{return err}
    if err:=hrCheck(comCall(link,7,uintptr(unsafe.Pointer(utf16("Launch Natro Editor with NatroMacro folder sync.")))),"SetDescription");err!=nil{return err}
    // The real Natro Editor.exe has a built-in yarn ball icon.
    if err:=hrCheck(comCall(link,17,uintptr(unsafe.Pointer(utf16(target))),0),"SetIconLocation");err!=nil{return err}
    p,err:=persistFile(link);if err!=nil{return err};defer comCall(p,2)
    return hrCheck(comCall(p,6,uintptr(unsafe.Pointer(utf16(linkPath))),1),"Save shortcut")
}
func safeShortcut(linkPath,target string) error {
    if _,err:=os.Stat(linkPath);err==nil {
        prev,err:=readShortcutTarget(linkPath)
        if err!=nil {return fmt.Errorf("cannot verify shortcut: %w",err)}
        previousName:=strings.ToLower(filepath.Base(prev))
        if !strings.EqualFold(filepath.Clean(prev),filepath.Clean(target)) &&
          previousName!="natro editor.exe" && previousName!="hivepath_studio.exe" {return fmt.Errorf("refusing to overwrite unrelated shortcut %s",linkPath)}
    } else if !errors.Is(err,os.ErrNotExist) {return err}
    // Refresh same-target links too: Windows icon cache can hold the old globe.
    return writeShortcut(linkPath,target)
}
func desktopFolder() (string,error) {
    var result uintptr
    hr,_,_:=shGetKnownFolderPath.Call(uintptr(unsafe.Pointer(&folderIDDesktop)),0,0,uintptr(unsafe.Pointer(&result)))
    if err:=hrCheck(hr,"SHGetKnownFolderPath");err!=nil{return "",err}
    if result==0{return "",errors.New("Desktop path empty")};defer coTaskMemFree.Call(result)
    chars:=make([]uint16,0,260)
    for i:=0;i<32767;i++ {
        c:=*(*uint16)(unsafe.Pointer(result+uintptr(i*2)))
        if c==0{break};chars=append(chars,c)
    }
    return syscall.UTF16ToString(chars),nil
}
func retireLegacyShortcut(desktop,target string) {
    for _,name:=range []string{"Natro Editor (Natro Sync).lnk","HivePath Studio (Natro Sync).lnk"} {
        old:=filepath.Join(desktop,name)
        if _,err:=os.Stat(old);err!=nil{continue}
        path,err:=readShortcutTarget(old)
        if err!=nil{continue}
        if !strings.EqualFold(filepath.Dir(path),filepath.Dir(target)){continue}
        base:=strings.ToLower(filepath.Base(path))
        if base!="natro editor.exe" && base!="hivepath_studio.exe"{continue}
        _=os.Remove(old)
    }
}
func ensureDesktopShortcut(target string) {
    desktop,err:=desktopFolder();if err!=nil{logError("Desktop lookup: "+err.Error());return}
    link:=filepath.Join(desktop,"Natro Editor.lnk")
    if err:=safeShortcut(link,target);err!=nil{logError("Shortcut: "+err.Error());return}
    retireLegacyShortcut(desktop,target)
    // Broadcast shell icon refresh: no registry change.
    shChangeNotify.Call(0x08000000,0,0,0) // SHCNE_ASSOCCHANGED
}
func launch(path string) error {
    r,_,_:=shellExecuteW.Call(0,uintptr(unsafe.Pointer(utf16("open"))),uintptr(unsafe.Pointer(utf16(path))),0,
        uintptr(unsafe.Pointer(utf16(filepath.Dir(path)))),1)
    if r<=32{return fmt.Errorf("ShellExecute failed (code %d): %s",r,path)}
    return nil
}
func portable() error {
    local:=os.Getenv("LOCALAPPDATA");if local==""{local=os.TempDir()}
    target:=filepath.Join(local,"HivePathStudio","Portable","Natro Editor.exe")
    if err:=verifiedWrite(target,coreExe);err!=nil{return err}
    ensureDesktopShortcut(target)
    return launch(target)
}
func run() error {
    path,err:=os.Executable();if err!=nil{return err}
    path,err=filepath.Abs(path);if err!=nil{return err}
    home,_:=os.UserHomeDir()
    root:=scanForNatro(path,home,readStoredRoot(),nil,6*time.Second)
    if root=="" {logError("No unique NatroMacro folder detected; starting portable editor");return portable()}
    target:=filepath.Join(root,"Natro Editor.exe")
    if strings.EqualFold(path,target){return portable()} // never overwrite a running launcher
    if err:=verifiedWrite(target,coreExe);err!=nil {
        logError("Install skipped: "+err.Error())
        if isNatroEditorExecutable(target){ensureDesktopShortcut(target);return launch(target)}
        return portable()
    }
    if err:=saveState(target);err!=nil{logError("State: "+err.Error())}
    if err:=safeShortcut(strings.TrimSuffix(path,filepath.Ext(path))+".lnk",target);err!=nil{logError("Local shortcut: "+err.Error())}
    ensureDesktopShortcut(target)
    return launch(target)
}
func main() {if err:=run();err!=nil{logError("Startup: "+err.Error())}}
