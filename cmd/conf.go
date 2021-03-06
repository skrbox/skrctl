package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
)

type conf struct {
	// 纳管的服务列表
	services []*service
	// 管控目录，之后可能有更多用途
	controlDir string
	workDir    string

	locker *sync.Mutex
}

// 从本地 .skrctl 目录读取
func newConf() *conf {
	workDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("加载本地配置失败: %s\n", err.Error())
		os.Exit(1)
	}
	dir := path.Join(workDir, ".skrctl")
	c := conf{workDir: workDir, controlDir: dir, locker: new(sync.Mutex)}
	stat, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.Mkdir(dir, 0755); err != nil {
				fmt.Println("初始化配置失败:", err.Error())
				os.Exit(1)
			}
			return &c
		}
		fmt.Printf("加载本地配置失败: %s\n", err.Error())
		os.Exit(1)
	}
	if !stat.IsDir() {
		fmt.Println("配置文件不合法")
		os.Exit(1)
	}
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		fmt.Printf("加载本地配置失败: %s\n", err.Error())
		os.Exit(1)
	}
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".service" {
			c.services = append(c.services, newService(file.Name(), path.Join(dir, file.Name())))
		}
	}
	return &c
}

// 服务是否已经被纳管
func (c *conf) Has(name string) *service {
	name = strings.TrimSuffix(name, ".service")
	for _, existed := range c.services {
		if name == existed.Name {
			return existed
		}
	}
	return nil
}

// 已托管服务清单
func (c *conf) List() []*service {
	return c.services
}

// 增加service托管
func (c *conf) Add(file string) error {
	c.locker.Lock()
	defer c.locker.Unlock()
	if filepath.Ext(file) != ".service" {
		return fmt.Errorf("必须添加一个合法service文件")
	}
	stat, err := os.Stat(file)
	if err != nil {
		return err
	}
	if stat.IsDir() {
		return fmt.Errorf("必须添加一个合法service文件")
	}
	if c.Has(stat.Name()) != nil {
		return fmt.Errorf("%s已存在", stat.Name())
	}
	expected := path.Join(c.controlDir, stat.Name())
	err = cp(file, expected)
	if err != nil {
		return err
	}
	c.services = append(c.services, newService(stat.Name(), expected))
	return nil
}

// 移除service托管(只管理本地配置)
func (c *conf) Remove(name string) {
	c.locker.Lock()
	defer c.locker.Unlock()
	name = strings.TrimSuffix(name, ".service")
	svc := c.Has(name)
	if svc == nil {
		return
	}
	_ = os.Remove(svc.File)
	var index int
	for i, svc := range c.services {
		if svc.Name == name {
			index = i
		}
	}
	c.services = append(c.services[:index], c.services[index+1:]...)
}

// 工作目录
func (c *conf) WorkDir() string {
	return c.workDir
}

// 控制目录
func (c *conf) ControlDir() string {
	return c.controlDir
}
