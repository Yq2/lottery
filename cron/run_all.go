
package cron

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
)

/**
 * 需要每个应用服务器都要运行的服务
 * 单个应用的服务
 */
const pprof_cpu_file  = "../debug/cpu.prof"
const pprof_mem_file  = "../debug/mem.out"
const pprof_block_file  = "../debug/block.out"
const mem_profile_rate  = 512 * 1024  //每分配指定的字节数量后对内存使用情况进行取样,默认512K
const block_profile_rate   = 1 //每发生几次Goroutine阻塞事件时对这些事件进行取样，默认1

type ProfileType string

func ConfigueAppAllCron() {
	 go pprofRun()
}

func pprofRun() {
	startCPUProfile()
	defer stopCPUProfile()
	startMemProfile()
	defer stopMemProfile()
	stopBlockProfile()
	defer stopBlockProfile()
	SaveProfile("../debug/","goroutine.out","goroutine",1)
	SaveProfile("../debug/","threadcreate.out","threadcreate",1)
}

func startCPUProfile() {
	if pprof_cpu_file != "" {
		f, err := os.OpenFile(pprof_cpu_file, os.O_RDWR|os.O_CREATE, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Can not create cpu profile output file: %s",
				err)
			return
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			fmt.Fprintf(os.Stderr, "Can not start cpu profile: %s", err)
			f.Close()
			return
		}
	}
}

func stopCPUProfile() {
	if pprof_cpu_file != "" {
		pprof.StopCPUProfile() // 把记录的概要信息写到已指定的文件
	}
}

func startMemProfile() {
	if pprof_mem_file != "" && mem_profile_rate > 0 {
		runtime.MemProfileRate = mem_profile_rate
	}
}

func stopMemProfile() {
	if pprof_mem_file != "" {
		fm, err := os.OpenFile(pprof_mem_file, os.O_RDWR|os.O_CREATE, 0644)
		if err != nil {
			log.Fatal(err)
		}
		pprof.WriteHeapProfile(fm)
		fm.Close()
	}
}

func startBlockProfile() {
	if pprof_block_file != "" && block_profile_rate > 0 {
		runtime.SetBlockProfileRate(block_profile_rate)
	}
}

func stopBlockProfile() {
	if pprof_block_file != "" &&  block_profile_rate >=0 {
		f, err := os.OpenFile(pprof_block_file, os.O_RDWR|os.O_CREATE, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Can not create block profile output file: %s", err)
			return
		}
		if err = pprof.Lookup("block").WriteTo(f, 0); err != nil {
			fmt.Fprintf(os.Stderr, "Can not write %s: %s", pprof_block_file, err)
		}
		f.Close()
	}
}

func SaveProfile(workDir string, profileName string, ptype ProfileType, debug int) {
	//workDir 文件存放的目录
	//profileName 概要文件的名称，必须为 "goroutine","threadcreate","heap","block"中的一个
	//ptype 概要文件的类型
	//debug = 0,1,2
	absWorkDir := getAbsFilePath(workDir)
	if profileName == "" {
		profileName = string(ptype)
	}
	profilePath := filepath.Join(absWorkDir, profileName)
	//f, err := os.Create(profilePath)
	f, err := os.OpenFile(profilePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can not create profile output file: %s", err)
		return
	}
	if err = pprof.Lookup(string(ptype)).WriteTo(f, debug); err != nil {
		fmt.Fprintf(os.Stderr, "Can not write %s: %s", profilePath, err)
	}
	f.Close()
}

func getAbsFilePath(dir string) string{
	fpt, err := filepath.Abs(dir)
	if err != nil {
		panic(errors.New(fmt.Sprintf("getAbsFilePath.%s",err)))
	}
	return fpt
}
