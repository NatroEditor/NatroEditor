package main

import (
    "context"
    "os"
    "path/filepath"
    "sort"
    "strings"
    "time"
)

type scanRoot struct { path string; depth int; bonus int }
type natroMatch struct { path string; score int }
var ignoredDirectories=map[string]bool{
    ".git":true,".svn":true,"node_modules":true,"$recycle.bin":true,
    "windows":true,"program files":true,"program files (x86)":true,
    "appdata":true,"application data":true,"system volume information":true,
    "venv":true,".venv":true,"__pycache__":true,"cache":true,
    ".cache":true,".gradle":true,".minecraft":true,"tmp":true,
    "temp":true,".next":true,"target":true,
}
func looksLikeNatroRoot(path string) (int,bool) {
    p,err:=os.Stat(filepath.Join(path,"patterns"))
    if err!=nil || !p.IsDir(){return 0,false}
    name:=strings.ToLower(filepath.Base(path))
    n:=strings.NewReplacer("-","","_",""," ","").Replace(name)
    score:=0
    if n=="natromacro" || n=="natromacromain"{score+=180}
    if strings.Contains(n,"natro"){score+=100}
    for _,marker:=range []string{"NatroMacro.ahk","Natro Macro.ahk","Natro_Macro.ahk","Natro.ahk"} {
        if file,err:=os.Stat(filepath.Join(path,marker));err==nil && !file.IsDir(){score+=200;break}
    }
    if entries,err:=os.ReadDir(filepath.Join(path,"patterns"));err==nil {
        for _,entry:=range entries {
            if !entry.IsDir() && strings.EqualFold(filepath.Ext(entry.Name()),".ahk") {score+=35;break}
        }
    }
    if score<100 {return 0,false}
    lower:=strings.ToLower(filepath.ToSlash(path))
    for _,hint:=range []string{"/backup","/archive","/old/","/copy of"} {
        if strings.Contains(lower,hint){score-=65;break}
    }
    return score,true
}
func scanForNatro(launcherExe,home,storedRoot string,additional []scanRoot,timeout time.Duration) string {
    roots:=[]scanRoot{}
    add:=func(p string,depth,bonus int){if p!="" {roots=append(roots,scanRoot{p,depth,bonus})}}
    launcherDir:=filepath.Dir(launcherExe)
    add(launcherDir,4,40)
    add(filepath.Dir(launcherDir),2,15)
    if storedRoot!=""{add(storedRoot,0,300)}
    if home!=""{
        add(home,1,0)
        for _,name:=range []string{"Desktop","Downloads","Documents","OneDrive","OneDrive - Personal","OneDrive - Business","Games"} {
            add(filepath.Join(home,name),5,3)
        }
    }
    for _,e:=range []string{"OneDrive","OneDriveConsumer","OneDriveCommercial"}{add(os.Getenv(e),5,3)}
    for _,drive:=range []string{"C:","D:","E:"}{add(drive+`\`,2,0)}
    roots=append(roots,additional...)
    ctx,cancel:=context.WithTimeout(context.Background(),timeout);defer cancel()
    type item struct{path string;depth int;bonus int}
    queue:=make([]item,0,512)
    for _,root:=range roots{queue=append(queue,item{root.path,root.depth,root.bonus})}
    seen:=make(map[string]bool)
    matches:=[]natroMatch{}
    scanned:=0
    for len(queue)>0 && scanned<14000 {
        select {case <-ctx.Done():queue=nil;continue;default:}
        it:=queue[0];queue=queue[1:]
        abs,err:=filepath.Abs(it.path);if err!=nil{continue}
        key:=strings.ToLower(filepath.Clean(abs));if seen[key]{continue};seen[key]=true
        stat,err:=os.Lstat(abs)
        if err!=nil || !stat.IsDir() || stat.Mode()&os.ModeSymlink!=0{continue}
        scanned++
        if score,ok:=looksLikeNatroRoot(abs);ok{matches=append(matches,natroMatch{abs,score+it.bonus})}
        if it.depth<=0{continue}
        children,err:=os.ReadDir(abs);if err!=nil{continue}
        for _,c:=range children {
            if !c.IsDir() || c.Type()&os.ModeSymlink!=0{continue}
            n:=strings.ToLower(c.Name())
            if ignoredDirectories[n] || strings.HasPrefix(n,".") || strings.HasSuffix(n,".app"){continue}
            queue=append(queue,item{filepath.Join(abs,c.Name()),it.depth-1,it.bonus})
        }
    }
    if len(matches)==0 {return ""}
    sort.Slice(matches,func(i,j int)bool{
        if matches[i].score==matches[j].score{return strings.ToLower(matches[i].path)<strings.ToLower(matches[j].path)}
        return matches[i].score>matches[j].score
    })
    if len(matches)>1 && matches[0].score==matches[1].score{return ""}
    return matches[0].path
}
