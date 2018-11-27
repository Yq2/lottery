
package cron

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
)

/**
 * 需要每个应用服务器都要运行的服务
 * 单个应用的服务
 */
const pprof_cpu_file  = "./cpu.prof"
const pprof_mem_file  = "./mem.out"
const pprof_block_file  = "./block.out"
const mem_profile_rate  = 512 * 1024  //每分配指定的字节数量后对内存使用情况进行取样,默认512K
const block_profile_rate   = 5 //每发生几次Goroutine阻塞事件时对这些事件进行取样，默认1
 
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