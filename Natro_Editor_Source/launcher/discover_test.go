package main
import (
  "os"
  "path/filepath"
  "testing"
  "time"
)
func TestDiscovery(t *testing.T){
 dir:=t.TempDir();root:=filepath.Join(dir,"NatroMacro")
 if err:=os.MkdirAll(filepath.Join(root,"patterns"),0755);err!=nil{t.Fatal(err)}
 if err:=os.WriteFile(filepath.Join(root,"patterns","sample.ahk"),[]byte("#NoEnv\n"),0644);err!=nil{t.Fatal(err)}
 if _,ok:=looksLikeNatroRoot(root);!ok{t.Fatal("expected NatroMacro patterns")}
 found:=scanForNatro(filepath.Join(dir,"NatroEditor.exe"),"","",[]scanRoot{{root,0,300}},time.Second)
 if filepath.Clean(found)!=filepath.Clean(root){t.Fatalf("expected %q got %q",root,found)}
}
